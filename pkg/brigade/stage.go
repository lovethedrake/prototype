package brigade

import (
	"log"
	"sync"

	"github.com/lovethedrake/prototype/pkg/config"
)

func (e *executor) runStage(
	project Project,
	event Event,
	pipelineName string,
	stageIndex int,
	targets []config.Target,
) error {
	log.Printf("executing pipeline \"%s\" stage %d", pipelineName, stageIndex)
	errCh := make(chan error)
	wg := &sync.WaitGroup{}
	for _, target := range targets {
		log.Printf(
			"executing pipeline \"%s\" stage %d target \"%s\"",
			pipelineName,
			stageIndex,
			target.Name(),
		)
		wg.Add(1)
		go e.runTargetPod(
			project,
			event,
			pipelineName,
			stageIndex,
			target,
			wg,
			errCh,
		)
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()
	errs := []error{}
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 1 {
		return &multiError{errs: errs}
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return nil
}
