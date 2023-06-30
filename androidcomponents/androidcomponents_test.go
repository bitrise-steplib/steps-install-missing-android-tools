package androidcomponents

import (
	"io"
	"os/exec"
	"testing"

	commandv2 "github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-steplib/steps-install-missing-android-tools/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
