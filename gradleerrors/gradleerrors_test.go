package gradleerrors

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type BuildFailure struct {
	BuildSlug    string `json:"build_slug"`
	BuildNumber  string `json:"build_number"`
	ErrorMessage string `json:"error_message"`
}

//go:embed bquxjob_3bd703a9_188e2535d5f.json
var failuresSrc string

func readBuildFailures(failuresSrc string) ([]BuildFailure, error) {
	var buildFailures []BuildFailure
	if err := json.Unmarshal([]byte(failuresSrc), &buildFailures); err != nil {
		return nil, err
	}
	return buildFailures, nil
}

func TestGradleErrors(t *testing.T) {
	failures, err := readBuildFailures(failuresSrc)
	require.NoError(t, err)
	for _, failure := range failures {
		if failure.BuildSlug == "e227f6d0-e2dc-4f14-9f49-72169383f65c" ||
			failure.BuildSlug == "aa209734-ecb1-4d02-aead-2da3dd0013f0" ||
			failure.BuildSlug == "bea9e4d0-feed-4bc8-8896-0ff8bedbbbd3" ||
			failure.BuildSlug == "3cb5cc49-d41a-47ee-96e8-a4fa00215cf5" { // get back to this 'compileKotlin FAILED'
			continue
		}

		if !strings.HasPrefix(failure.ErrorMessage, "Run: failed to ensure android components") {
			continue
		}

		failure.ErrorMessage = strings.TrimPrefix(failure.ErrorMessage, "Run: failed to ensure android components, error: output: ")
		failure.ErrorMessage = strings.TrimSuffix(failure.ErrorMessage, "error: exit status 1")

		if failure.BuildSlug == "748a7557-72e3-4d9f-8b8e-1889940a4d0c" {
			fmt.Println(failure.BuildSlug)
		}
		fmt.Println(failure.BuildSlug)
		detectedCausedByErrors, err := ErrorCausedByFinder{}.findErrors(failure.ErrorMessage)
		fmt.Println(detectedCausedByErrors)
		require.NoError(t, err, failure.BuildSlug)

		detectedFailures, err := FailureFinder{}.findErrors(failure.ErrorMessage)
		fmt.Println(detectedFailures)
		require.NoError(t, err, failure.BuildSlug)

		require.True(t, len(detectedCausedByErrors) > 0 || len(detectedFailures) > 0, failure.BuildSlug)
		require.False(t, len(detectedCausedByErrors) > 0 && len(detectedFailures) > 0, failure.BuildSlug)

	}
}

func TestMultipleFailuresFinder(t *testing.T) {
	out := `FAILURE: Build completed with 2 failures.

1: Task failed with an exception.
-----------
* Where:
Build file '/bitrise/src/app/build.gradle' line: 14

* What went wrong:
A problem occurred evaluating project ':app'.
> /bitrise/src/apikey.properties (No such file or directory)

* Try:
> Run with --info or --debug option to get more log output.
> Run with --scan to get full insights.

* Exception is:
org.gradle.api.GradleScriptException: A problem occurred evaluating project ':app'.
	at org.gradle.groovy.scripts.internal.DefaultScriptRunnerFactory$ScriptRunnerImpl.run(DefaultScriptRunnerFactory.java:93)
	at org.gradle.configuration.DefaultScriptPluginFactory$ScriptPluginImpl.lambda$apply$0(DefaultScriptPluginFactory.java:133)

==============================================================================

2: Task failed with an exception.
-----------
* What went wrong:
A problem occurred configuring project ':app'.
> compileSdkVersion is not specified. Please add it to build.gradle

* Try:
> Run with --info or --debug option to get more log output.
> Run with --scan to get full insights.

* Exception is:
org.gradle.api.ProjectConfigurationException: A problem occurred configuring project ':app'.
	at org.gradle.configuration.project.LifecycleProjectEvaluator.wrapException(LifecycleProjectEvaluator.java:75)
`
	reasons, err := MultipleFailuresFinder{}.findErrors(out)
	require.NoError(t, err)
	require.Equal(t, reasons, []string{})
}
