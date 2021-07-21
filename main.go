package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bitrise-io/go-android/sdk"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-install-missing-android-tools/androidcomponents"
	"github.com/hashicorp/go-version"
)

const androidNDKHome = "ANDROID_NDK_HOME"

// Config ...
type Config struct {
	GradlewPath    string `env:"gradlew_path,file"`
	AndroidHome    string `env:"ANDROID_HOME"`
	AndroidSDKRoot string `env:"ANDROID_SDK_ROOT"`
	NDKRevision    string `env:"ndk_revision"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func ndkDownloadURL(revision string) string {
	return fmt.Sprintf("https://dl.google.com/android/repository/android-ndk-r%s-%s-x86_64.zip", revision, runtime.GOOS)
}

func installedNDKVersion(ndkHome string) string {
	propertiesPath := filepath.Join(ndkHome, "source.properties")

	content, err := fileutil.ReadStringFromFile(propertiesPath)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(strings.ToLower(line), "pkg.revision") {
			lineParts := strings.Split(line, "=")
			if len(lineParts) == 2 {
				revision := strings.TrimSpace(lineParts[1])
				version, err := version.NewVersion(revision)
				if err != nil {
					return ""
				}
				return fmt.Sprintf("%d", version.Segments()[0])
			}
		}
	}
	return ""
}

func ndkHome() string {
	if v := os.Getenv(androidNDKHome); v != "" {
		return v
	}
	if v := os.Getenv("ANDROID_HOME"); v != "" {
		return filepath.Join(v, "android-ndk-bundle")
	}
	if v := os.Getenv("ANDROID_SDK_ROOT"); v != "" {
		return filepath.Join(v, "android-ndk-bundle")
	}
	if v := os.Getenv("HOME"); v != "" {
		return filepath.Join(v, "android-ndk-bundle")
	}
	return "android-ndk-bundle"
}

func inPath(path string) bool {
	return strings.Contains(os.Getenv("PATH"), path)
}

func updateNDK(revision string) error {
	ndkURL := ndkDownloadURL(revision)
	ndkHome := ndkHome()

	if currentRevision := installedNDKVersion(ndkHome); currentRevision == revision {
		log.Donef("NDK r%s already installed", revision)
		return nil
	}

	log.Printf("NDK home: %s", ndkHome)
	log.Printf("Cleaning")

	if err := os.RemoveAll(ndkHome); err != nil {
		return err
	}

	if err := pathutil.EnsureDirExist(ndkHome); err != nil {
		return err
	}

	log.Printf("Downloading")

	if err := command.DownloadAndUnZIP(ndkURL, ndkHome); err != nil {
		return err
	}

	if !inPath(ndkHome) {
		log.Printf("Append to $PATH")
		if err := tools.ExportEnvironmentWithEnvman("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), ndkHome)); err != nil {
			return err
		}
	}

	if os.Getenv(androidNDKHome) == "" {
		log.Printf("Export %s: %s", androidNDKHome, ndkHome)
		if err := tools.ExportEnvironmentWithEnvman(androidNDKHome, ndkHome); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Input validation
	var config Config
	if err := stepconf.Parse(&config); err != nil {
		log.Errorf("%s", err)
	}

	fmt.Println()
	stepconf.Print(config)

	fmt.Println()
	log.Infof("Preparation")

	// Set executable permission for gradlew
	log.Printf("Set executable permission for gradlew")
	if err := os.Chmod(config.GradlewPath, 0770); err != nil {
		failf("Failed to set executable permission for gradlew, error: %s", err)
	}

	fmt.Println()
	if config.NDKRevision != "" {
		log.Infof("Installing NDK bundle")

		if err := updateNDK(config.NDKRevision); err != nil {
			failf("Failed to download NDK bundle, error: %s", err)
		}
	} else {
		log.Infof("Clearing NDK environment")
		log.Printf("Unset ANDROID_NDK_HOME")

		if err := os.Unsetenv("ANDROID_NDK_HOME"); err != nil {
			failf("Failed to unset environment variable, error: %s", err)
		}

		if err := tools.ExportEnvironmentWithEnvman("ANDROID_NDK_HOME", ""); err != nil {
			failf("Failed to set environment variable, error: %s", err)
		}
	}

	// Initialize Android SDK
	log.Printf("Initialize Android SDK")
	androidSdk, err := sdk.NewDefaultModel(sdk.Environment{
		AndroidHome:    config.AndroidHome,
		AndroidSDKRoot: config.AndroidSDKRoot,
	})
	if err != nil {
		failf("Failed to initialize Android SDK: %s", err)
	}

	androidcomponents.SetLogger(log.NewDefaultLogger(false))

	// Ensure android licences
	log.Printf("Ensure android licences")

	if err := androidcomponents.InstallLicences(androidSdk); err != nil {
		failf("Failed to ensure android licences, error: %s", err)
	}

	// Ensure required Android SDK components
	fmt.Println()
	log.Infof("Ensure required Android SDK components")

	if err := androidcomponents.Ensure(androidSdk, config.GradlewPath); err != nil {
		failf("Failed to ensure android components, error: %s", err)
	}

	fmt.Println()
	log.Donef("Required SDK components are installed")
}
