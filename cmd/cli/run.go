package main

import (
	"context"
	"path/filepath"

	docker "github.com/docker/docker/client"
	drakecli "github.com/lovethedrake/prototype/pkg/cli"
	drakedocker "github.com/lovethedrake/prototype/pkg/orchestration/docker"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func run(c *cli.Context) error {
	configFile := c.GlobalString(flagFile)
	debugOnly := c.Bool(flagDebug)
	concurrencyEnabled := c.Bool(flagConcurrently)
	absConfigFilePath, err := filepath.Abs(configFile)
	if err != nil {
		return err
	}
	sourcePath := filepath.Dir(absConfigFilePath)
	dockerClient, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		return errors.Wrap(err, "error building Docker client")
	}
	executor := drakecli.NewExecutor(drakedocker.NewOrchestrator(dockerClient))
	executePipelines := c.Bool(flagPipeline)
	if executePipelines {
		if len(c.Args()) == 0 {
			return errors.New("no pipelines were specified for execution")
		}
		// TODO: Should pass the stream that we want output to go to-- stdout
		// TODO: Make this context cancelable; probably need to do some signal
		// handling
		err = executor.ExecutePipelines(
			context.Background(),
			configFile,
			sourcePath,
			c.Args(),
			debugOnly,
			concurrencyEnabled,
		)
	} else {
		if len(c.Args()) == 0 {
			return errors.New("no targets were specified for execution")
		}
		// TODO: Should pass the stream that we want output to go to-- stdout
		// TODO: Make this context cancelable; probably need to do some signal
		// handling
		err = executor.ExecuteTargets(
			context.Background(),
			configFile,
			sourcePath,
			c.Args(),
			debugOnly,
			concurrencyEnabled,
		)
	}
	return err
}
