package gradle_wrapper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindAll_WhenMultipleWrappersExist_FindsAllWrappers(t *testing.T) {
	root := t.TempDir()

	wrappers := []string{
		filepath.Join(root, "app", "gradlew"),
		filepath.Join(root, "app", "gradlew.bat"),
		filepath.Join(root, "modules", "feature", "gradlew"),
	}

	items := []struct {
		path    string
		isDir   bool
		perm    os.FileMode
		content []byte
	}{
		{path: filepath.Join(root, "app", "gradlew"), perm: 0o755, content: []byte("#!/bin/sh\n")},
		{path: filepath.Join(root, "app", "gradlew.bat"), perm: 0o755, content: []byte("#!/bin/sh\n")},
		{path: filepath.Join(root, "modules", "feature", "gradlew"), perm: 0o755, content: []byte("#!/bin/sh\n")},
		{path: filepath.Join(root, "empty"), isDir: true, perm: 0o755},
		{path: filepath.Join(root, "modules", "empty-nested"), isDir: true, perm: 0o755},
		{path: filepath.Join(root, "README.md"), perm: 0o644, content: []byte("")},
		{path: filepath.Join(root, "app", "gradle-wrapper.properties"), perm: 0o644, content: []byte("")},
		{path: filepath.Join(root, "modules", "feature", "build.gradle"), perm: 0o644, content: []byte("")},
	}

	for _, item := range items {
		if item.isDir {
			require.NoError(t, os.MkdirAll(item.path, item.perm))
			continue
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(item.path), 0o755))
		require.NoError(t, os.WriteFile(item.path, item.content, item.perm))
	}

	found, err := FindAll(root)
	require.NoError(t, err)
	require.ElementsMatch(t, wrappers, found)
}
