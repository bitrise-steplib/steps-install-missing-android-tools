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
