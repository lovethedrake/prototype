package brigade

import (
	"log"
	"sync"

	"github.com/lovethedrake/prototype/pkg/config"
)

func (e *executor) runPipeline(
	project Project,
	event Event,
	pipeline config.Pipeline,
	wg *sync.WaitGroup,
	errCh chan<- error,
) {
	defer wg.Done()
	log.Printf("executing pipeline \"%s\"", pipeline.Name())
	log.Printf("creating shared storage for pipeline \"%s\"", pipeline.Name())
	if err := e.createSrcPVC(project, event, pipeline.Name()); err != nil {
		errCh <- err
		return
	}
	log.Printf("created shared storage for pipeline \"%s\"", pipeline.Name())
	defer func() {
		log.Printf("destroying shared storage for pipeline \"%s\"", pipeline.Name())
		if err := e.destroySrcPVC(project, event, pipeline.Name()); err != nil {
			log.Printf(
				"error destroying shared storage for pipeline \"%s\": %s",
				pipeline.Name(),
				err,
			)
			return
		}
		log.Printf("destroyed shared storage for pipeline \"%s\"", pipeline.Name())
	}()
	log.Printf(
		"cloning source to shared storage for pipeline \"%s\"",
		pipeline.Name(),
	)
	if err := e.runSourceClonePod(project, event, pipeline.Name()); err != nil {
		errCh <- err
		return
	}
	log.Printf(
		"cloned source to shared storage for pipeline \"%s\"",
		pipeline.Name(),
	)
	for stageIndex, stageTargets := range pipeline.Targets() {
		if err := e.runStage(
			project,
			event,
			pipeline.Name(),
			stageIndex,
			stageTargets,
		); err != nil {
			errCh <- err
			return
		}
	}
}
