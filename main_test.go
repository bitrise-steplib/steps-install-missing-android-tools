package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_currentNDKHome(t *testing.T) {
	type test struct {
		name string
		envs map[string]string
		wantPath string
		wantCleanup bool
	}

	requestedNDKVersion := "23.0.7599858"
	tests := []test{
		{
			name: "ANDROID_NDK_HOME is set",
			envs: map[string]string{
				"ANDROID_NDK_HOME": "/opt/android-ndk",
			},
			wantPath: "/opt/android-ndk",
			wantCleanup: true,
		},
		{
			name: "both ANDROID_NDK_HOME and ANDROID_HOME are set",
			envs: map[string]string{
				"ANDROID_NDK_HOME": "/opt/android-ndk",
				"ANDROID_HOME": "/opt/android-sdk",
			},
			wantPath: "/opt/android-ndk",
			wantCleanup: true,
		},
		{
			name: "only ndk-bundle is installed",
			envs: map[string]string{
				"ANDROID_HOME": "/home/user/android-sdk",
			},
			wantPath: "/home/user/android-sdk/ndk/23.0.7599858",
			wantCleanup: false,
		},
		{
			name: "ndk-bundle and side-by-side NDK is installed",
			envs: map[string]string{
				"ANDROID_HOME": "/home/user/android-sdk",
			},
			wantPath: "/home/user/android-sdk/ndk/23.0.7599858",
			wantCleanup: false,
		},
		{
			name: "the exact requested side-by-side NDK is installed",
			envs: map[string]string{
				"ANDROID_HOME": "/home/user/android-sdk",
			},
			wantPath: "/home/user/android-sdk/ndk/23.0.7599858",
			wantCleanup: false,
		},
		{
			name: "a different side-by-side NDK is installed than requested",
			envs: map[string]string{
				"ANDROID_HOME": "/home/user/android-sdk",
			},
			wantPath: "/home/user/android-sdk/ndk/23.0.7599858",
			wantCleanup: false,
		},
		{
			name: "both ANDROID_HOME and SDK_ROOT are set, pointing to the same dir",
			envs: map[string]string{
				"ANDROID_HOME": "/home/user/android-sdk",
				"SDK_ROOT": "/home/user/android-sdk",
			},
			wantPath: "/home/user/android-sdk/ndk/23.0.7599858",
			wantCleanup: false,
		},
		{
			name: "both ANDROID_HOME and SDK_ROOT are set, pointing to different dirs",
			envs: map[string]string{
				"ANDROID_HOME": "/home/user/android-sdk-home",
				"ANDROID_SDK_ROOT": "/home/user/android-sdk-root",
			},
			wantPath: "/home/user/android-sdk-home/ndk/23.0.7599858",
			wantCleanup: false,
		},
		{
			name: "only SDK_ROOT is set",
			envs: map[string]string{
				"ANDROID_SDK_ROOT": "/home/user/android-sdk-root",
			},
			wantPath: "/home/user/android-sdk-root/ndk/23.0.7599858",
			wantCleanup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envRepo := fakeEnvRepo{envVars: tt.envs}
			gotPath, gotCleanup := targetNDKPath(envRepo, requestedNDKVersion)
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
