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
			return NewCommandExitError(cmd, exitErr)
		}

		return NewCommandExitErrorWithReason(cmd, exitErr, reason)

	}
	return NewCommandExecutionError(cmd, err)
}

type CommandExecutionError struct {
	cmd string
	err error
}

func NewCommandExecutionError(cmd string, err error) CommandExecutionError {
	return CommandExecutionError{
		cmd: cmd,
		err: err,
	}
}

func (e CommandExecutionError) Error() string {
	return fmt.Sprintf("executing command failed (%s): %s", e.cmd, e.err)
}

func (e CommandExecutionError) Unwrap() error {
	return e.err
}

type CommandExitError struct {
	cmd        string
	err        *exec.ExitError
	suggestion error
}

func NewCommandExitError(cmd string, err *exec.ExitError) CommandExitError {
	return CommandExitError{
		cmd:        cmd,
		err:        err,
		suggestion: errors.New("check the command's output for details"),
	}
}

func (e CommandExitError) Error() string {
	return fmt.Sprintf("command failed with exit status %d (%s): %s", e.err.ExitCode(), e.cmd, e.suggestion)
}

func (e CommandExitError) Unwrap() error {
	return e.suggestion
}

type CommandExitErrorWithReason struct {
	cmd    string
	err    *exec.ExitError
	reason error
}

func NewCommandExitErrorWithReason(cmd string, err *exec.ExitError, reason string) CommandExitErrorWithReason {
	return CommandExitErrorWithReason{
		cmd:    cmd,
		err:    err,
		reason: errors.New(reason),
	}
}

func (e CommandExitErrorWithReason) Error() string {
	return fmt.Sprintf("command failed with exit status %d (%s): %s", e.err.ExitCode(), e.cmd, e.reason)
}

func (e CommandExitErrorWithReason) Unwrap() error {
	return e.reason
}
