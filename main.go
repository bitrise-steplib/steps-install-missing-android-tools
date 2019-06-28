package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bitrise-steplib/steps-install-missing-android-tools/androidcomponents"
	"github.com/bitrise-tools/go-steputils/input"
	"github.com/bitrise-tools/go-steputils/tools"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-android/sdk"
)

const platformDirName = "platforms"
const androidNDKHome = "ANDROID_NDK_HOME"

// ConfigsModel ...
type ConfigsModel struct {
	GradlewPath string
	AndroidHome string
	NDKRevision string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		GradlewPath: os.Getenv("gradlew_path"),
		AndroidHome: os.Getenv("ANDROID_HOME"),
		NDKRevision: os.Getenv("ndk_revision"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- GradlewPath: %s", configs.GradlewPath)
	log.Printf("- AndroidHome: %s", configs.AndroidHome)
	log.Printf("- NDKRevision: %s", configs.NDKRevision)
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfPathExists(configs.GradlewPath); err != nil {
		return errors.New("Issue with input GradlewPath: " + err.Error())
	}

	if err := input.ValidateIfNotEmpty(configs.AndroidHome); err != nil {
		return errors.New("Issue with input AndroidHome: " + err.Error())
	}

	return nil
}

// -----------------------
// --- Functions
// -----------------------

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

	if err := updateNDKPathIfNeeded(ndkHome); err != nil {
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

func updateNDKPathIfNeeded(ndkHome string) error {
	log.Printf("searching for platforms dir in %s", ndkHome)
	var foundPDPath string
	if err := filepath.Walk(ndkHome, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == platformDirName {
			foundPDPath = path.Dir(currentPath)
			log.Printf("found platforms dir at %s", foundPDPath)
			if foundPDPath != ndkHome {
				return updateNDKPath(foundPDPath)
			}
			log.Printf("%s is a valid NDK dir, no need to update", foundPDPath)
			return nil
		}
		return nil
	}); err != nil {
		return err
	}
	if foundPDPath == "" {
		return fmt.Errorf("could not locate platform dir at ndk home: %s", ndkHome)
	}
	return nil
}

func updateNDKPath(path string) error {
	if err := os.Setenv(androidNDKHome, path); err != nil {
		return fmt.Errorf("failed to add path key %s with value %s", androidNDKHome, path)
	}
	if err := tools.ExportEnvironmentWithEnvman(androidNDKHome, path); err != nil {
		return fmt.Errorf("failed to export with envman key %s value %s", androidNDKHome, path)
	}
	log.Printf("updated NDK HOME path to %s", path)
	return nil
}

// -----------------------
// --- Main
// -----------------------

func main() {
	// Input validation
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		failf(err.Error())
	}

	fmt.Println()
	log.Infof("Preparation")

	// Set executable permission for gradlew
	log.Printf("Set executable permission for gradlew")
	if err := os.Chmod(configs.GradlewPath, 0770); err != nil {
		failf("Failed to set executable permission for gradlew, error: %s", err)
	}

	fmt.Println()
	if configs.NDKRevision != "" {
		log.Infof("Installing NDK bundle")

		if err := updateNDK(configs.NDKRevision); err != nil {
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
	androidSdk, err := sdk.New(configs.AndroidHome)
	if err != nil {
		failf("Failed to initialize Android SDK, error: %s", err)
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

	if err := androidcomponents.Ensure(androidSdk, configs.GradlewPath); err != nil {
		failf("Failed to ensure android components, error: %s", err)
	}

	fmt.Println()
	log.Donef("Required SDK components are installed")
}
