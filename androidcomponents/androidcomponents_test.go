package androidcomponents

import (
	"errors"
	"io"
	"os/exec"
	"testing"

	commandv2 "github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-steplib/steps-install-missing-android-tools/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_GivenCommand_WhenFails_ThenReturnsExitError(t *testing.T) {
	// TODO: androidcomponents.NewCommandError requires a command execution function to return an *exec.ExitError,
	// when the command was successfully executed, but returned non-zero exit status.
	// In go-utils/v2@v2.0.0-alpha.15 the command package was updated to return a new custom error.
	// Upgrading to this or higher version breaks androidcomponents.NewCommandError.
	// This test ensures that the used go-utils/v2/command package works well with androidcomponents.NewCommandError.
	factory := commandv2.NewFactory(env.NewRepository())
	cmd := factory.Create("bash", []string{"-c", "exit 1"}, nil)
	err := cmd.Run()
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
}

func Test_GivenInstallerAndGradlePrintsToStderr_WhenScanDependencies_ThenErrorContainStderr(t *testing.T) {
	// Given
	var stderr io.Writer

	command := new(mocks.Command)
	command.On("Run").Run(func(args mock.Arguments) {
		_, err := stderr.Write([]byte("error reason"))
		require.NoError(t, err)
	}).Return(&exec.ExitError{})
	command.On("PrintableCommandArgs").Return("./gradlew dependencies --stacktrace")

	factory := new(mocks.Factory)
	factory.On("Create", "./gradlew", []string{"dependencies", "--stacktrace"}, mock.Anything).Run(func(args mock.Arguments) {
		opts := args.Get(2).(*commandv2.Opts)
		stderr = opts.Stderr
	}).Return(command)

	installer := installer{
		gradlewPath: "./gradlew",
		factory:     factory,
	}

	// When
	err := installer.scanDependencies(false)

	// Then
	require.EqualError(t, err, "command failed with exit status -1 (./gradlew dependencies --stacktrace): error reason")
}

func Test_GivenInstallerAndGradleDoesNotPrintToStderr_WhenScanDependenciesAndLastAttempt_ThenErrorGenericErrorThrownAndStdoutLogged(t *testing.T) {
	// Given
	var stdout io.Writer

	command := new(mocks.Command)
	command.On("Run").Run(func(args mock.Arguments) {
		_, err := stdout.Write([]byte("Task failed"))
		require.NoError(t, err)
	}).Return(&exec.ExitError{})
	command.On("PrintableCommandArgs").Return("./gradlew dependencies --stacktrace")

	factory := new(mocks.Factory)
	factory.On("Create", "./gradlew", []string{"dependencies", "--stacktrace"}, mock.Anything).Run(func(args mock.Arguments) {
		opts := args.Get(2).(*commandv2.Opts)
		stdout = opts.Stdout
	}).Return(command)

	logger := new(mocks.Logger)
	logger.On("Printf", "Task failed").Return()

	installer := installer{
		gradlewPath: "./gradlew",
		factory:     factory,
		logger:      logger,
	}

	// When
	err := installer.scanDependencies(true)

	// Then
	require.EqualError(t, err, "command failed with exit status -1 (./gradlew dependencies --stacktrace): check the command's output for details")
	logger.AssertExpectations(t)
}
