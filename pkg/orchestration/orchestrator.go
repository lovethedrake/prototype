package orchestration

import (
	"context"

	"github.com/lovethedrake/prototype/pkg/config"
)

type Orchestrator interface {
	ExecuteTarget(
		ctx context.Context,
		executionName string,
		sourcePath string,
		target config.Target,
	) error
}
