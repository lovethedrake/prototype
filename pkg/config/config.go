package config

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// Config is a public interface for the root of the Drake configuration tree.
type Config interface {
	// GetTargets returns an ordered list of Target given the provided
	// targetNames.
	GetTargets(targetNames []string) ([]Target, error)
	// GetAllPipelines returns a list of all Pipelines
	GetAllPipelines() []Pipeline
	// GetPipelines returns an ordered list of Pipelines given the provided
	// pipelineNames.
	GetPipelines(pipelineNames []string) ([]Pipeline, error)
}

// config represents the root of the Drake configuration tree.
type config struct {
	Targets   map[string]*target   `json:"targets"`
	Pipelines map[string]*pipeline `json:"pipelines"`
}

// NewConfigFromFile loads configuration from the specified path and returns it.
func NewConfigFromFile(configFilePath string) (Config, error) {
	configFileBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil,
			errors.Wrapf(err, "error reading config file %s", configFilePath)
	}
	config := &config{}
	if err := yaml.Unmarshal(configFileBytes, config); err != nil {
		return nil,
			errors.Wrapf(err, "error unmarshalling config file %s", configFilePath)
	}
	// Step through all targets to add their name attribute, which is inferred
	// from the keys in the config.Targets map.
	for targetName, target := range config.Targets {
		target.name = targetName
	}
	// Step through all pipelines to add their name attribute, which is inferred
	// from the keys in the config.Pipelines map. Also resolve targets referenced
	// by each pipeline.
	for pipelineName, pipeline := range config.Pipelines {
		pipeline.name = pipelineName
		if err := pipeline.resolveTargets(config.Targets); err != nil {
			return nil, errors.Wrapf(
				err,
				"error resolving targets for pipeline \"%s\"",
				pipeline.name,
			)
		}
	}
	return config, nil
}

func (c *config) GetTargets(
	targetNames []string,
) ([]Target, error) {
	targets := []Target{}
	for _, targetName := range targetNames {
		target, ok := c.Targets[targetName]
		if !ok {
			return nil,
				errors.Errorf("target \"%s\" not found", targetName)
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func (c *config) GetAllPipelines() []Pipeline {
	pipelines := make([]Pipeline, len(c.Pipelines))
	var i int
	for _, pipeline := range c.Pipelines {
		pipelines[i] = pipeline
		i++
	}
	return pipelines
}

func (c *config) GetPipelines(
	pipelineNames []string,
) ([]Pipeline, error) {
	pipelines := []Pipeline{}
	for _, pipelineName := range pipelineNames {
		pipeline, ok := c.Pipelines[pipelineName]
		if !ok {
			return nil,
				errors.Errorf("pipeline \"%s\" not found", pipelineName)
		}
		pipelines = append(pipelines, pipeline)
	}
	return pipelines, nil
}
