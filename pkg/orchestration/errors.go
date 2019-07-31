package orchestration

import (
	"fmt"
)

type ErrJobExitedNonZero struct {
	Job      string
	ExitCode int64
}

func (e *ErrJobExitedNonZero) Error() string {
	return fmt.Sprintf(
		`job "%s" failed with non-zero exit code %d`,
		e.Job,
		e.ExitCode,
	)
}
