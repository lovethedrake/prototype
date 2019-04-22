package config

import "github.com/pkg/errors"

// Pipeline is a public interface for pipeline configuration.
type Pipeline interface {
	Name() string
	Matches(branch, tag string) (bool, error)
	Targets() [][]Target
}

type pipeline struct {
	name     string
	Criteria *pipelineExecutionCriteria `json:"criteria"`
	Stages   []*pipelineStage           `json:"stages"`
	targets  [][]*target
}

type pipelineStage struct {
	Targets []string `json:"targets"`
}

func (p *pipeline) resolveTargets(targets map[string]*target) error {
	p.targets = make([][]*target, len(p.Stages))
	for i, stage := range p.Stages {
		p.targets[i] = make([]*target, len(stage.Targets))
		for j, targetName := range stage.Targets {
			target, ok := targets[targetName]
			if !ok {
				return errors.Errorf(
					"pipeline \"%s\" stage %d (zero-indexed) depends on undefined "+
						"target \"%s\"",
					p.name,
					i,
					targetName,
				)
			}
			p.targets[i][j] = target
		}
	}
	return nil
}

func (p *pipeline) Name() string {
	return p.name
}

func (p *pipeline) Matches(branch, tag string) (bool, error) {
	// If no criteria are specified, the default is to match
	if p.Criteria == nil {
		return true, nil
	}
	return p.Criteria.Matches(branch, tag)
}

func (p *pipeline) Targets() [][]Target {
	targets := make([][]Target, len(p.targets))
	for i, targs := range p.targets {
		targets[i] = make([]Target, len(targs))
		for j, targ := range targs {
			targets[i][j] = targ
		}
	}
	return targets
}
