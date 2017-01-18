package analyzer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIncludedModules(t *testing.T) {
	t.Log("")
	{
		content := `include ':app', ':dynamicgrid'`

		modules, err := ParseIncludedModules(content)
		require.NoError(t, err)
		require.Equal(t, 2, len(modules))
		require.Equal(t, "app", modules[0])
		require.Equal(t, "dynamicgrid", modules[1])
	}
}

func TestParseCompileSDKVersion(t *testing.T) {
	t.Log("simple")
	{
		content := `
android {
    compileSdkVersion 23
    buildToolsVersion "23.0.3"
}
`
		v, err := parseCompileSDKVersion(content)
		require.NoError(t, err)
		require.NotNil(t, v)
		require.Equal(t, "23.0.0", v.String())
	}

	t.Log("no compileSdkVersion")
	{
		content := `
android {
    buildToolsVersion "23.0.3"
}
`
		v, err := parseCompileSDKVersion(content)
		require.Error(t, err)
		require.Nil(t, v)
	}
}

func TestParseBuildToolsVersion(t *testing.T) {
	t.Log("simple")
	{
		content := `
android {
    compileSdkVersion 23
    buildToolsVersion "23.0.3"
}
`
		v, err := parseBuildToolsVersion(content)
		require.NoError(t, err)
		require.NotNil(t, v)
		require.Equal(t, "23.0.3", v.String())
	}

	t.Log("no compileSdkVersion")
	{
		content := `
android {
	compileSdkVersion 23
}
`
		v, err := parseBuildToolsVersion(content)
		require.Error(t, err)
		require.Nil(t, v)
	}
}

func TestParseBuildGradle(t *testing.T) {
	t.Log("SDK21 + Tools21.0.1 + support")
	{
		deps, err := parseBuildGradle(testBuildGradleSDK21Tools2101FileContent)
		require.NoError(t, err)
		require.Equal(t, "21.0.0", deps.ComplieSDKVersion.String())
		require.Equal(t, "21.0.1", deps.BuildToolsVersion.String())
		require.Equal(t, true, deps.UseSupportLibrary)
		require.Equal(t, false, deps.UseGooglePlayServices)
	}

	t.Log("SDK24 + Tools24.0.2 + support")
	{
		deps, err := parseBuildGradle(testBuildGradleSDK24Tools2402SupportFileContent)
		require.NoError(t, err)
		require.Equal(t, "24.0.0", deps.ComplieSDKVersion.String())
		require.Equal(t, "24.0.2", deps.BuildToolsVersion.String())
		require.Equal(t, true, deps.UseSupportLibrary)
		require.Equal(t, false, deps.UseGooglePlayServices)
	}

	t.Log("SDK23 + Tools23.0.3 + support + play")
	{
		deps, err := parseBuildGradle(testBuildGradleSDK23Tools2303SupportPlayFileContent)
		require.NoError(t, err)
		require.Equal(t, "23.0.0", deps.ComplieSDKVersion.String())
		require.Equal(t, "23.0.3", deps.BuildToolsVersion.String())
		require.Equal(t, true, deps.UseSupportLibrary)
		require.Equal(t, true, deps.UseGooglePlayServices)
	}
}
