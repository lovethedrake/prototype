package orchestration

import (
	"fmt"
)

type ErrStepExitedNonZero struct {
	Target   string
	ExitCode int64
}

func (e *ErrStepExitedNonZero) Error() string {
	return fmt.Sprintf(
		`target "%s" failed with non-zero exit code %d`,
		e.Target,
		e.ExitCode,
	)
}
