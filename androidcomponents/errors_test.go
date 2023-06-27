package androidcomponents

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandExitErrorIsExitError(t *testing.T) {
	exitError := &exec.ExitError{}
	commandExitError := CommandExitError{
		cmd: "gradle dependencies",
		err: exitError,
	}

	var exitErr *exec.ExitError
	isExitErr := errors.As(commandExitError, &exitErr)
	require.True(t, isExitErr)
}

func TestCommandExitErrorWithReasonIsExitError(t *testing.T) {
	exitError := &exec.ExitError{}
	commandExitError := CommandExitError{
		cmd: "gradle dependencies",
		err: exitError,
	}
	commandExitErrorWithReason := CommandExitErrorWithReason{
		CommandExitError: commandExitError,
		reason: `Error: Could not find or load main class org.gradle.wrapper.GradleWrapperMain
Caused by: java.lang.ClassNotFoundException: org.gradle.wrapper.GradleWrapperMain`,
	}

	var exitErr *exec.ExitError
	isExitErr := errors.As(commandExitErrorWithReason, &exitErr)
	require.True(t, isExitErr)
}
