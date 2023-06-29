package androidcomponents

import (
	"errors"
	"fmt"
	"os/exec"
)

func NewCommandError(cmd string, err error, reason string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if len(reason) == 0 {
			return newCommandExitError(cmd, exitErr)
		}

		return newCommandExitErrorWithReason(cmd, exitErr, reason)
	}

	return newCommandExecutionError(cmd, err)
}

func newCommandExecutionError(cmd string, err error) error {
	return fmt.Errorf("executing command failed (%s): %w", cmd, err)
}

func newCommandExitError(cmd string, err *exec.ExitError) error {
	suggestion := errors.New("check the command's output for details")
	return fmt.Errorf("command failed with exit status %d (%s): %w", err.ExitCode(), cmd, suggestion)
}

func newCommandExitErrorWithReason(cmd string, err *exec.ExitError, reasonStr string) error {
	reason := errors.New(reasonStr)
	return fmt.Errorf("command failed with exit status %d (%s): %w", err.ExitCode(), cmd, reason)
}
