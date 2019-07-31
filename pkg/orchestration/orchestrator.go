package orchestration

import (
	"context"

	"github.com/lovethedrake/prototype/pkg/config"
)

type Orchestrator interface {
	ExecuteJob(
		ctx context.Context,
		secrets []string,
		executionName string,
		sourcePath string,
		job config.Job,
		errCh chan<- error,
	)
}
