package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	dockerTypes "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/lovethedrake/prototype/pkg/config"
	"github.com/lovethedrake/prototype/pkg/orchestration"
	shellwords "github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
)

type devOrchestrator struct {
	dockerClient *docker.Client
}

func NewOrchestrator(dockerClient *docker.Client) orchestration.Orchestrator {
	return &devOrchestrator{
		dockerClient: dockerClient,
	}
}

func (d *devOrchestrator) ExecuteTarget(
	ctx context.Context,
	targetExecutionName string,
	sourcePath string,
	target config.Target,
) error {
	if len(target.Containers()) == 0 {
		return nil
	}
	containerIDs := make([]string, len(target.Containers()))
	// Ensure cleanup of all containers
	defer d.forceRemoveContainers(ctx, containerIDs...)
	var networkContainerID, lastContainerID string
	var lastContainer config.Container
	// Create and start all containers-- except the last one-- that one we will
	// only create, then we will set ourselves up to capture its output and exit
	// code before we start it.
	for i, container := range target.Containers() {
		containerID, err := d.createContainer(
			ctx,
			targetExecutionName,
			sourcePath,
			networkContainerID,
			container,
		)
		if err != nil {
			return errors.Wrapf(
				err,
				"error creating container \"%s\" for target \"%s\"",
				container.Name(),
				target.Name(),
			)
		}
		containerIDs[i] = containerID
		if i == 0 {
			networkContainerID = containerID
		}
		if i == len(containerIDs)-1 {
			lastContainerID = containerID
			lastContainer = container
		} else {
			// Start all but the last container
			if err := d.dockerClient.ContainerStart(
				ctx,
				containerID,
				dockerTypes.ContainerStartOptions{},
			); err != nil {
				return errors.Wrapf(
					err,
					"error starting container \"%s\" for target \"%s\"",
					container.Name(),
					target.Name(),
				)
			}
		}
	}
	// Establish channels to use for waiting for the last container to exit
	containerWaitRespCh, containerWaitErrCh := d.dockerClient.ContainerWait(
		ctx,
		lastContainerID,
		dockerContainer.WaitConditionNextExit,
	)
	// Attach to the last container to see its output
	containerAttachResp, err := d.dockerClient.ContainerAttach(
		ctx,
		lastContainerID,
		dockerTypes.ContainerAttachOptions{
			Stream: true,
			Stdout: true,
			Stderr: true,
		},
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error attaching to container \"%s\" for target \"%s\"",
			lastContainer.Name(),
			target.Name(),
		)
	}
	// Concurrently deal with the output from the last container
	go func() {
		defer containerAttachResp.Close()
		var gerr error
		if lastContainer.TTY() {
			_, gerr = io.Copy(os.Stdout, containerAttachResp.Reader)
		} else {
			_, gerr = stdcopy.StdCopy(
				os.Stdout,
				os.Stderr,
				containerAttachResp.Reader,
			)
		}
		if gerr != nil {
			fmt.Printf(
				"error processing output from container \"%s\" for target \"%s\": %s\n",
				lastContainer.Name(),
				target.Name(),
				err,
			)
		}
	}()
	// Finally, start the last container
	if err := d.dockerClient.ContainerStart(
		ctx,
		lastContainerID,
		dockerTypes.ContainerStartOptions{},
	); err != nil {
		return errors.Wrapf(
			err,
			"error starting container \"%s\" for target \"%s\"",
			lastContainer.Name(),
			target.Name(),
		)
	}
	select {
	case containerWaitResp := <-containerWaitRespCh:
		if containerWaitResp.StatusCode != 0 {
			// The command executed inside the container exited non-zero
			return &orchestration.ErrStepExitedNonZero{
				Target:   target.Name(),
				ExitCode: containerWaitResp.StatusCode,
			}
		}
	case err := <-containerWaitErrCh:
		return errors.Wrapf(
			err,
			"error waiting for completion of container \"%s\" for target \"%s\"",
			lastContainer.Name(),
			target.Name(),
		)
	}
	return nil
}

// createContainer creates a container for the given execution and target,
// taking source path, any established networking, and container-specific
// configuration into account. It returns the newly created container's ID. It
// does not start the container.
func (d *devOrchestrator) createContainer(
	ctx context.Context,
	targetExecutionName string,
	sourcePath string,
	networkContainerID string,
	container config.Container,
) (string, error) {
	containerConfig := &dockerContainer.Config{
		Image:        container.Image(),
		Env:          container.Environment(),
		WorkingDir:   container.WorkingDirectory(),
		Tty:          container.TTY(),
		AttachStdout: true,
		AttachStderr: true,
	}
	if container.Command() != "" {
		cmd, err := shellwords.Parse(container.Command())
		if err != nil {
			return "", errors.Wrap(err, "error parsing container command")
		}
		containerConfig.Cmd = cmd
	}
	hostConfig := &dockerContainer.HostConfig{
		Privileged: container.Privileged(),
	}
	if networkContainerID != "" {
		hostConfig.NetworkMode = dockerContainer.NetworkMode(
			fmt.Sprintf("container:%s", networkContainerID),
		)
	}
	if container.MountDockerSocket() {
		hostConfig.Binds = []string{"/var/run/docker.sock:/var/run/docker.sock"}
	}
	if container.SourceMountPath() != "" {
		hostConfig.Binds = append(
			hostConfig.Binds,
			fmt.Sprintf("%s:%s", sourcePath, container.SourceMountPath()),
		)
	}
	fullContainerName := fmt.Sprintf(
		"%s-%s",
		targetExecutionName,
		container.Name(),
	)
	containerCreateResp, err := d.dockerClient.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		fullContainerName,
	)
	if err != nil {
		return "",
			errors.Wrapf(
				err,
				"error creating container \"%s\"",
				fullContainerName,
			)
	}
	return containerCreateResp.ID, nil
}

func (d *devOrchestrator) forceRemoveContainers(
	ctx context.Context,
	containerIDs ...string,
) {
	for _, containerID := range containerIDs {
		if err := d.dockerClient.ContainerRemove(
			ctx,
			containerID,
			dockerTypes.ContainerRemoveOptions{
				Force: true,
			},
		); err != nil {
			// TODO: Maybe this isn't the best way to deal with this
			fmt.Printf(`error removing container "%s": %s`, containerID, err)
		}
	}
}
