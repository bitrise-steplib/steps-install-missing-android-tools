package main

import (
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func Test_currentNDKHome(t *testing.T) {
	type test struct {
		name string
		envs map[string]string
		fs   fs.FS
		wantPath string
		wantCleanup bool
	}

	requestedNDKVersion := "23.0.7599858"
	// Note: the initial / is omitted in all paths below because of the limitation of fstest.MapFS:
	// https://github.com/golang/go/issues/51378
	tests := []test{
		{
			name: "ANDROID_NDK_HOME is set",
			envs: map[string]string{
				"ANDROID_NDK_HOME": "opt/android-ndk",
			},
			fs: fstest.MapFS{
				"opt/android-ndk/source.properties": &fstest.MapFile{
					Data: []byte("Pkg.Revision = " + requestedNDKVersion),
				},
			},
			wantPath: "opt/android-ndk",
			wantCleanup: true,
		},
		{
			name: "only ndk-bundle is installed",
			envs: map[string]string{
				"ANDROID_HOME": "home/user/android-sdk",
			},
			fs: fstest.MapFS{
				"home/user/android-sdk/ndk-bundle/source.properties": &fstest.MapFile{
					Data: []byte("Pkg.Revision = 22.1.7171670"),
				},
			},
			wantPath: "home/user/android-sdk/ndk/23.0.7599858",
			wantCleanup: false,
		},
		{
			name: "ndk-bundle and side-by-side NDK is installed",
			envs: map[string]string{
				"ANDROID_HOME": "home/user/android-sdk",
			},
			fs: fstest.MapFS{
				"home/user/android-sdk/ndk-bundle/source.properties": &fstest.MapFile{
					Data: []byte("Pkg.Revision = 22.1.7171670"),
				},
				"home/user/android-sdk/ndk/23.0.7599858/source.properties": &fstest.MapFile{
					Data: []byte("Pkg.Revision = " + requestedNDKVersion),
				},
			},
			wantPath: "home/user/android-sdk/ndk/23.0.7599858",
			wantCleanup: false,
		},
		{
			name: "the exact requested side-by-side NDK is installed",
			envs: map[string]string{
				"ANDROID_HOME": "home/user/android-sdk",
			},
			fs: fstest.MapFS{
				"home/user/android-sdk/ndk/" + requestedNDKVersion + "/source.properties": &fstest.MapFile{
					Data: []byte("Pkg.Revision = " + requestedNDKVersion),
				},
			},
			wantPath: "home/user/android-sdk/ndk/23.0.7599858",
			wantCleanup: false,
		},
		{
			name: "a different side-by-side NDK is installed than requested",
			envs: map[string]string{
				"ANDROID_HOME": "home/user/android-sdk",
			},
			fs: fstest.MapFS{
				"home/user/android-sdk/ndk/22.1.7171670/source.properties": &fstest.MapFile{
					Data: []byte("Pkg.Revision = 22.1.7171670"),
				},
			},
			wantPath: "home/user/android-sdk/ndk/23.0.7599858",
			wantCleanup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envRepo := fakeEnvRepo{envVars: tt.envs}
			gotPath, gotCleanup := targetNDKPath(envRepo, tt.fs, requestedNDKVersion)
			require.Equal(t, tt.wantPath, gotPath)
			require.Equal(t, tt.wantCleanup, gotCleanup)
		})
	}

}

type fakeEnvRepo struct {
	envVars map[string]string
}

func (repo fakeEnvRepo) Get(key string) string {
	value, ok := repo.envVars[key]
	if ok {
		return value
	} else {
		return ""
	}
}

func (repo fakeEnvRepo) Set(key, value string) error {
	repo.envVars[key] = value
	return nil
}

func (repo fakeEnvRepo) Unset(key string) error {
	repo.envVars[key] = ""
	return nil
}

func (repo fakeEnvRepo) List() []string {
	envs := []string{}
	for k, v := range repo.envVars {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}
