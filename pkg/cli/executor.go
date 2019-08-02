package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	dockerTypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/lovethedrake/drakecore/config"
	"github.com/lovethedrake/prototype/pkg/orchestration"
	"github.com/technosophos/moniker"
)

// Executor is the public interface for the CLI executor
type Executor interface {
	ExecuteJobs(
		ctx context.Context,
		configFile string,
		secretsFile string,
		sourcePath string,
		jobNames []string,
		debugOnly bool,
		concurrencyEnabled bool,
	) error
	ExecutePipelines(
		ctx context.Context,
		configFile string,
		secretsFile string,
		sourcePath string,
		pipelineNames []string,
		debugOnly bool,
		concurrencyEnabled bool,
	) error
}

type executor struct {
	namer        moniker.Namer
	orchestrator orchestration.Orchestrator
	dockerClient *docker.Client
}

// NewExecutor returns an executor suitable for use with local development
func NewExecutor(
	dockerClient *docker.Client,
	orchestrator orchestration.Orchestrator,
) Executor {
	return &executor{
		namer:        moniker.New(),
		orchestrator: orchestrator,
		dockerClient: dockerClient,
	}
}

func (e *executor) ExecuteJobs(
	ctx context.Context,
	configFile string,
	secretsFile string,
	sourcePath string,
	jobNames []string,
	debugOnly bool,
	concurrencyEnabled bool,
) error {
	config, err := config.NewConfigFromFile(configFile)
	if err != nil {
		return err
	}
	secrets, err := secretsFromFile(secretsFile)
	if err != nil {
		return err
	}
	jobs, err := config.Jobs(jobNames)
	if err != nil {
		return err
	}
	if debugOnly {
		fmt.Printf("would execute jobs: %s\n", jobNames)
		return nil
	}

	imageNames := map[string]struct{}{}
	for _, job := range jobs {
		for _, container := range job.Containers() {
			imageNames[container.Image()] = struct{}{}
		}
	}
	for imageName := range imageNames {
		fmt.Printf("~~~~> pulling image \"%s\" <~~~~\n", imageName)
		reader, err := e.dockerClient.ImagePull(
			ctx,
			imageName,
			dockerTypes.ImagePullOptions{},
		)
		if err != nil {
			return err
		}
		defer reader.Close()
		dec := json.NewDecoder(reader)
		for {
			var message jsonmessage.JSONMessage
			if err := dec.Decode(&message); err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			fmt.Println(message.Status)
		}
	}

	executionName := e.namer.NameSep("-")
	errCh := make(chan error)
	var runningJobs int
	for _, job := range jobs {
		jobExecutionName := fmt.Sprintf("%s-%s", executionName, job.Name())
		runningJobs++
		go e.orchestrator.ExecuteJob(
			ctx,
			secrets,
			jobExecutionName,
			sourcePath,
			job,
			errCh,
		)
		if !concurrencyEnabled {
			// If concurrency isn't enabled, wait for a potential error. If it's nil,
			// move on. If it's not, return the error.
			if err := <-errCh; err != nil {
				return err
			}
			runningJobs--
		}
	}
	// If concurrency isn't enabled and we haven't already encountered an error,
	// then we're not going to. We're done!
	if !concurrencyEnabled {
		return nil
	}
	// Wait for all the jobs to finish.
	errs := []error{}
	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
		runningJobs--
		if runningJobs == 0 {
			break
		}
	}
	if len(errs) > 1 {
		return &multiError{errs: errs}
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return nil
}

func (e *executor) ExecutePipelines(
	ctx context.Context,
	configFile string,
	secretsFile string,
	sourcePath string,
	pipelineNames []string,
	debugOnly bool,
	concurrencyEnabled bool,
) error {
	config, err := config.NewConfigFromFile(configFile)
	if err != nil {
		return err
	}
	secrets, err := secretsFromFile(secretsFile)
	if err != nil {
		return err
	}
	pipelines, err := config.Pipelines(pipelineNames)
	if err != nil {
		return err
	}
	if debugOnly {
		fmt.Println("would execute:")
		for _, pipeline := range pipelines {
			jobs := make([][]string, len(pipeline.Jobs()))
			for i, stageJobs := range pipeline.Jobs() {
				jobs[i] = make([]string, len(stageJobs))
				for j, job := range stageJobs {
					jobs[i][j] = job.Name()
				}
			}
			fmt.Printf("  %s jobs: %s\n", pipeline.Name(), jobs)
		}
		return nil
	}

	imageNames := map[string]struct{}{}
	for _, pipeline := range pipelines {
		for _, stageJobs := range pipeline.Jobs() {
			for _, job := range stageJobs {
				for _, container := range job.Containers() {
					imageNames[container.Image()] = struct{}{}
				}
			}
		}
	}
	for imageName := range imageNames {
		fmt.Printf("~~~~> pulling image \"%s\" <~~~~\n", imageName)
		reader, err := e.dockerClient.ImagePull(
			ctx,
			imageName,
			dockerTypes.ImagePullOptions{},
		)
		if err != nil {
			return err
		}
		defer reader.Close()
		dec := json.NewDecoder(reader)
		for {
			var message jsonmessage.JSONMessage
			if err := dec.Decode(&message); err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			fmt.Println(message.Status)
		}
	}

	executionName := e.namer.NameSep("-")
	for _, pipeline := range pipelines {
		fmt.Printf("====> executing pipeline \"%s\" <====\n", pipeline.Name())
		pipelineExecutionName :=
			fmt.Sprintf("%s-%s", executionName, pipeline.Name())
		for i, stageJobs := range pipeline.Jobs() {
			fmt.Printf("====> executing stage %d <====\n", i)
			stageExecutionName :=
				fmt.Sprintf("%s-stage%d", pipelineExecutionName, i)
			errCh := make(chan error)
			var runningJobs int
			for _, job := range stageJobs {
				jobExecutionName :=
					fmt.Sprintf("%s-%s", stageExecutionName, job.Name())
				runningJobs++
				go e.orchestrator.ExecuteJob(
					ctx,
					secrets,
					jobExecutionName,
					sourcePath,
					job,
					errCh,
				)
				// If concurrency isn't enabled, wait for a potential error. If it's
				// nil, move on. If it's not, return the error.
				if !concurrencyEnabled {
					if err := <-errCh; err != nil {
						return err
					}
					runningJobs--
				}
			}
			// If concurrency is enabled, wait for all the jobs to finish.
			if concurrencyEnabled {
				errs := []error{}
				for err := range errCh {
					if err != nil {
						errs = append(errs, err)
					}
					runningJobs--
					if runningJobs == 0 {
						break
					}
				}
				if len(errs) > 1 {
					return &multiError{errs: errs}
				}
				if len(errs) == 1 {
					return errs[0]
				}
			}
		}
	}
	return nil
}
