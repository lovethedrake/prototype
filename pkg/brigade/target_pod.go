package brigade

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lovethedrake/prototype/pkg/config"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	api "k8s.io/kubernetes/pkg/apis/core"
)

const (
	srcVolumeName          = "src"
	dockerSocketVolumeName = "docker-socket"
)

func (e *executor) runTargetPod(
	project Project,
	event Event,
	pipelineName string,
	stage int,
	target config.Target,
	wg *sync.WaitGroup,
	errCh chan<- error,
) {
	defer wg.Done()
	if err := notifyCheckStart(event, target.Name(), target.Name()); err != nil {
		errCh <- err
		return
	}
	conclusion := "failure"
	defer func() {
		if err := notifyCheckCompleted(
			event,
			target.Name(),
			target.Name(),
			conclusion,
		); err != nil {
			log.Printf("error sending notification to github: %s", err)
		}
	}()

	jobName := fmt.Sprintf("%s-stage%d-%s", pipelineName, stage, target.Name())
	podName := fmt.Sprintf("%s-%s", jobName, event.BuildID)
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"heritage":             "brigade",
				"component":            "job",
				"jobname":              jobName,
				"project":              project.ID,
				"worker":               event.WorkerID,
				"build":                event.BuildID,
				"thedrake.io/pipeline": pipelineName,
				"thedrake.io/stage":    strconv.Itoa(stage),
				"thedrake.io/target":   target.Name(),
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers:    []v1.Container{},
			Volumes: []v1.Volume{
				{
					Name: srcVolumeName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: srcPVCName(event.WorkerID, pipelineName),
						},
					},
				},
				{
					Name: dockerSocketVolumeName,
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/var/run/docker.sock",
						},
					},
				},
			},
		},
	}
	var mainContainerName string
	containers := target.Containers()
	for i, container := range containers {
		targetPodContainer, err := getTargetPodContainer(container)
		if err != nil {
			errCh <- err
			return
		}
		// We'll treat all but the last container as sidecars. i.e. The last
		// container in the target should be container 0 in the pod spec.
		if i < len(containers)-1 {
			pod.Spec.Containers = append(pod.Spec.Containers, targetPodContainer)
			continue
		}
		// This is the primary container. Make it the first (0th) in the pod spec.
		mainContainerName = container.Name()
		pod.Spec.Containers = append(
			[]v1.Container{targetPodContainer},
			pod.Spec.Containers...,
		)
	}

	_, err := e.kubeClient.CoreV1().Pods(project.Kubernetes.Namespace).Create(pod)
	if err != nil {
		errCh <- errors.Wrapf(err, "error creating pod \"%s\"", podName)
		return
	}

	podsWatcher, err :=
		e.kubeClient.CoreV1().Pods(project.Kubernetes.Namespace).Watch(
			metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(
					api.ObjectNameField,
					podName,
				).String(),
			},
		)
	if err != nil {
		errCh <- err
		return
	}

	// Timeout
	// TODO: This probably should not be hard-coded
	timer := time.NewTimer(10 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case event := <-podsWatcher.ResultChan():
			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				errCh <- errors.Errorf(
					"received unexpected object when watching pod \"%s\" for completion",
					podName,
				)
				return
			}
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Name == mainContainerName {
					if containerStatus.State.Terminated != nil {
						if containerStatus.State.Terminated.Reason == "Completed" {
							conclusion = "success"
							return
						}
						errCh <- errors.Errorf("pod \"%s\" failed", podName)
						return
					}
					break
				}
			}
		case <-timer.C:
			errCh <- errors.Errorf(
				"timed out waiting for pod \"%s\" to complete",
				podName,
			)
			conclusion = "timed_out"
			return
		}
	}
}

func getTargetPodContainer(container config.Container) (v1.Container, error) {
	privileged := container.Privileged()
	command, err := shellwords.Parse(container.Command())
	if err != nil {
		return v1.Container{}, err
	}
	c := v1.Container{
		Name:            container.Name(),
		Image:           container.Image(),
		ImagePullPolicy: v1.PullAlways,
		Command:         command,
		Env:             []v1.EnvVar{},
		SecurityContext: &v1.SecurityContext{
			Privileged: &privileged,
		},
		VolumeMounts: []v1.VolumeMount{},
		Stdin:        container.TTY(),
		TTY:          container.TTY(),
	}
	for _, kv := range container.Environment() {
		kvTokens := strings.SplitN(kv, "=", 2)
		if len(kvTokens) == 2 {
			c.Env = append(
				c.Env,
				v1.EnvVar{
					Name:  kvTokens[0],
					Value: kvTokens[1],
				},
			)
			continue
		}
		if len(kvTokens) == 1 {
			c.Env = append(
				c.Env,
				v1.EnvVar{
					Name: kvTokens[0],
				},
			)
		}
	}
	if container.SourceMountPath() != "" {
		c.VolumeMounts = append(
			c.VolumeMounts,
			v1.VolumeMount{
				Name:      srcVolumeName,
				MountPath: container.SourceMountPath(),
			},
		)
	}
	if container.WorkingDirectory() != "" {
		c.WorkingDir = container.WorkingDirectory()
	}
	if container.MountDockerSocket() {
		c.VolumeMounts = append(
			c.VolumeMounts,
			v1.VolumeMount{
				Name:      dockerSocketVolumeName,
				MountPath: "/var/run/docker.sock",
			},
		)
	}
	return c, nil
}
