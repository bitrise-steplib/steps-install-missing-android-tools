package androidcomponents

import (
	"errors"
	"fmt"
	"os/exec"
)

func NewCommandError(cmd string, err error, reason string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		commandExitErr := CommandExitError{
			cmd: cmd,
			err: exitErr,
		}

		if len(reason) == 0 {
			return commandExitErr
		}

		return CommandExitErrorWithReason{
			CommandExitError: commandExitErr,
			reason:           reason,
		}

	}
	return CommandExecutionError{
		cmd: cmd,
		err: err,
	}
}

type CommandExecutionError struct {
	cmd string
	err error
}

func (e CommandExecutionError) Error() string {
	return fmt.Sprintf("executing command failed (%s): %s", e.cmd, e.err)
}

func (e CommandExecutionError) Unwrap() error {
	return e.err
}

type CommandExitError struct {
	cmd string
	err *exec.ExitError
}

func (e CommandExitError) Error() string {
	suggestion := errors.New("check the command's output for details")
	return fmt.Sprintf("command failed with exit status %d (%s): %s", e.err.ExitCode(), e.cmd, suggestion)
}

func (e CommandExitError) Unwrap() error {
	return e.err
}

type CommandExitErrorWithReason struct {
	CommandExitError
	reason string
}

func (e CommandExitErrorWithReason) Error() string {
	return fmt.Sprintf("command failed with exit status %d (%s): %s", e.err.ExitCode(), e.cmd, e.reason)
}
