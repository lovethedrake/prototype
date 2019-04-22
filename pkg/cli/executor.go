package cli

import (
	"context"
	"fmt"

	"github.com/lovethedrake/prototype/pkg/config"
	"github.com/lovethedrake/prototype/pkg/orchestration"
	"github.com/technosophos/moniker"
)

// Executor is the public interface for the CLI executor
type Executor interface {
	ExecuteTargets(
		ctx context.Context,
		configFile string,
		sourcePath string,
		targetNames []string,
		debugOnly bool,
	) error
	ExecutePipelines(
		ctx context.Context,
		configFile string,
		sourcePath string,
		pipelineNames []string,
		debugOnly bool,
	) error
}

type executor struct {
	namer        moniker.Namer
	orchestrator orchestration.Orchestrator
}

// NewExecutor returns an executor suitable for use with local development
func NewExecutor(orchestrator orchestration.Orchestrator) Executor {
	return &executor{
		namer:        moniker.New(),
		orchestrator: orchestrator,
	}
}

func (e *executor) ExecuteTargets(
	ctx context.Context,
	configFile string,
	sourcePath string,
	targetNames []string,
	debugOnly bool,
) error {
	config, err := config.NewConfigFromFile(configFile)
	if err != nil {
		return err
	}
	targets, err := config.GetTargets(targetNames)
	if err != nil {
		return err
	}
	if debugOnly {
		fmt.Printf("would execute targets: %s\n", targetNames)
		return nil
	}
	executionName := e.namer.NameSep("-")
	for _, target := range targets {
		targetExecutionName := fmt.Sprintf("%s-%s", executionName, target.Name())
		if err := e.orchestrator.ExecuteTarget(
			ctx,
			targetExecutionName,
			sourcePath,
			target,
		); err != nil {
			return err
		}
	}
	return nil
}

func (e *executor) ExecutePipelines(
	ctx context.Context,
	configFile string,
	sourcePath string,
	pipelineNames []string,
	debugOnly bool,
) error {
	config, err := config.NewConfigFromFile(configFile)
	if err != nil {
		return err
	}
	pipelines, err := config.GetPipelines(pipelineNames)
	if err != nil {
		return err
	}
	if debugOnly {
		fmt.Println("would execute:")
		for _, pipeline := range pipelines {
			targets := make([][]string, len(pipeline.Targets()))
			for i, stageTargets := range pipeline.Targets() {
				targets[i] = make([]string, len(stageTargets))
				for j, target := range stageTargets {
					targets[i][j] = target.Name()
				}
			}
			fmt.Printf("  %s targets: %s\n", pipeline.Name(), targets)
		}
		return nil
	}
	executionName := e.namer.NameSep("-")
	for _, pipeline := range pipelines {
		pipelineExecutionName :=
			fmt.Sprintf("%s-%s", executionName, pipeline.Name())
		for i, stageTargets := range pipeline.Targets() {
			stageExecutionName :=
				fmt.Sprintf("%s-stage%d", pipelineExecutionName, i)
			for _, target := range stageTargets {
				targetExecutionName :=
					fmt.Sprintf("%s-%s", stageExecutionName, target.Name())
				if err := e.orchestrator.ExecuteTarget(
					ctx,
					targetExecutionName,
					sourcePath,
					target,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
