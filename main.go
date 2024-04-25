package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-android/sdk"
	"github.com/bitrise-io/go-android/sdkcomponent"
	"github.com/bitrise-io/go-android/sdkmanager"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/errorutil"
	. "github.com/bitrise-io/go-utils/v2/exitcode"
	"github.com/bitrise-io/go-utils/v2/log/colorstring"
	"github.com/bitrise-steplib/steps-install-missing-android-tools/androidcomponents"
	"github.com/hashicorp/go-version"
	"github.com/kballard/go-shellquote"
)

const androidNDKHome = "ANDROID_NDK_HOME"

type Inputs struct {
	GradlewPath                string `env:"gradlew_path,file"`
	AndroidHome                string `env:"ANDROID_HOME"`
	AndroidSDKRoot             string `env:"ANDROID_SDK_ROOT"`
	NDKVersion                 string `env:"ndk_version"`
	GradlewDependenciesOptions string `env:"gradlew_dependencies_options"`
}

type Config struct {
	GradlewPath                string
	AndroidHome                string
	AndroidSDKRoot             string
	NDKVersion                 string
	GradlewDependenciesOptions []string
}

func main() {
	exitCode := run()
	os.Exit(int(exitCode))
}

func run() ExitCode {
	androidToolsInstaller := AndroidToolsInstaller{}
	config, err := androidToolsInstaller.ProcessInputs()
	if err != nil {
		log.Errorf(errorutil.FormattedError(fmt.Errorf("Failed to process Step inputs: %w", err)))
		return Failure
	}

	if err := androidToolsInstaller.Run(config); err != nil {
		log.Errorf(errorutil.FormattedError(fmt.Errorf("Failed to execute Step: %w", err)))
		return Failure
	}

	return Success
}

type AndroidToolsInstaller struct {
}

func (i AndroidToolsInstaller) ProcessInputs() (Config, error) {
	var inputs Inputs
	if err := stepconf.Parse(&inputs); err != nil {
		return Config{}, err
	}
	gradlewDependenciesOptions, err := shellquote.Split(inputs.GradlewDependenciesOptions)
	if err != nil {
		return Config{}, fmt.Errorf("provided gradlew_dependencies_options (%s) are not valid CLI parameters: %s", inputs.GradlewDependenciesOptions, err)
	}

	config := Config{
		GradlewPath:                inputs.GradlewPath,
		AndroidHome:                inputs.AndroidHome,
		AndroidSDKRoot:             inputs.AndroidSDKRoot,
		NDKVersion:                 inputs.NDKVersion,
		GradlewDependenciesOptions: gradlewDependenciesOptions,
	}

	fmt.Println()
	stepconf.Print(config)

	return config, nil
}

func (i AndroidToolsInstaller) Run(config Config) error {
	fmt.Println()
	log.Infof("Preparation")

	// Set executable permission for gradlew
	log.Printf("Set executable permission for gradlew")
	if err := os.Chmod(config.GradlewPath, 0770); err != nil {
		return fmt.Errorf("failed to set executable permission for gradlew: %w", err)
	}

	// Initialize Android SDK
	fmt.Println()
	log.Infof("Initialize Android SDK")
	androidSdk, err := sdk.NewDefaultModel(sdk.Environment{
		AndroidHome:    config.AndroidHome,
		AndroidSDKRoot: config.AndroidSDKRoot,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize Android SDK: %w", err)
	}

	fmt.Println()
	if config.NDKVersion != "" {
		log.Infof("Installing Android NDK")

		_, err := version.NewVersion(config.NDKVersion)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid NDK version. This should be the full version number, such as 23.0.7599858. To see all available versions, run 'sdkmanager --list'", config.NDKVersion)
		}

		if err := updateNDK(config.NDKVersion, androidSdk); err != nil {
			return fmt.Errorf("install new NDK package: %w", err)
		}
	} else {
		log.Infof("Clearing NDK environment")
		log.Printf("Unset ANDROID_NDK_HOME")

		if err := os.Unsetenv("ANDROID_NDK_HOME"); err != nil {
			return fmt.Errorf("unset environment variable: %w", err)
		}

		if err := tools.ExportEnvironmentWithEnvman("ANDROID_NDK_HOME", ""); err != nil {
			return fmt.Errorf("failed to set environment variable: %w", err)
		}
	}

	// Ensure android licences
	log.Printf("Ensure android licences")

	if err := androidcomponents.InstallLicences(androidSdk); err != nil {
		return fmt.Errorf("failed to ensure android licences: %w", err)
	}

	// Ensure required Android SDK components
	fmt.Println()
	log.Infof("Ensure required Android SDK components")

	if err := androidcomponents.Ensure(androidSdk, config.GradlewPath, config.GradlewDependenciesOptions); err != nil {
		return fmt.Errorf("failed to install missing android components: %w", err)
	}

	fmt.Println()
	log.Donef("Required SDK components are installed")

	return nil
}

// ndkVersion returns the full version string of a given install path
func ndkVersion(ndkPath string) string {
	propertiesPath := filepath.Join(ndkPath, "source.properties")

	content, err := fileutil.ReadStringFromFile(propertiesPath)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(strings.ToLower(line), "pkg.revision") {
			lineParts := strings.Split(line, "=")
			if len(lineParts) == 2 {
				return strings.TrimSpace(lineParts[1])
			}
		}
	}
	return ""
}

// https://github.com/android/ndk-samples/wiki/Configure-NDK-Path
// https://developer.android.com/tools/variables
func targetNDKPath(envRepo env.Repository, sys fs.FS, requestedNDKVersion string) (string, bool) {
	if v := envRepo.Get(androidNDKHome); v != "" {
		// $ANDROID_NDK_HOME is old and AGP no longer takes it into account,
		// but it's an explicit path, so use it if it's set on the system.
		return v, true
	}
	if androidHome := envRepo.Get("ANDROID_HOME"); androidHome != "" {
		// The most modern way is to install NDK versions side-by-side at $ANDROID_HOME/ndk/version
		// This is what `sdkmanager` does when installing a specific version (`sdkmanager "ndk;26.3.11579264"`).
		ndkPath := filepath.Join(androidHome, "ndk", requestedNDKVersion)
		return ndkPath, false
	}
	if v := envRepo.Get("ANDROID_SDK_ROOT"); v != "" {
		// $ANDROID_SDK_ROOT is deprecated, so it's lower in priority than $ANDROID_HOME
		return filepath.Join(v, "ndk-bundle"), true
	}
	if v := envRepo.Get("HOME"); v != "" {
		return filepath.Join(v, "ndk-bundle"), true
	}
	return "ndk-bundle", true
}

// updateNDK installs the requested NDK version (if not already installed to the correct location).
// NDK is installed to the `ndk/version` subdirectory of the SDK location, while updating $ANDROID_NDK_HOME for
// compatibility with older Android Gradle Plugin versions.
// Details: https://github.com/android/ndk-samples/wiki/Configure-NDK-Path
func updateNDK(version string, androidSdk *sdk.Model) error {
	envRepo := env.NewRepository()
	targetNDKPath, doCleanup := targetNDKPath(envRepo, os.DirFS("/"), version)

	currentVersion := ndkVersion(targetNDKPath)
	if currentVersion == version {
		log.Donef("NDK %s already installed at %s", colorstring.Green(version), targetNDKPath)
		return nil
	}

	if currentVersion != "" || doCleanup {
		log.Printf("NDK %s found at: %s", colorstring.Cyan(currentVersion), targetNDKPath)
		log.Printf("Removing existing NDK...")
		if err := os.RemoveAll(targetNDKPath); err != nil {
			return err
		}
		log.Printf("Done")
	}

	log.Printf("Installing NDK %s with sdkmanager", colorstring.Cyan(version))
	sdkManager, err := sdkmanager.New(androidSdk)
	if err != nil {
		return err
	}
	ndkComponent := sdkcomponent.NDK{Version: version}
	cmd := sdkManager.InstallCommand(ndkComponent)
	output, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		log.Errorf(output)
		return fmt.Errorf("run %s: %w", cmd.PrintableCommandArgs(), err)
	}
	newNDKHome := filepath.Join(androidSdk.GetAndroidHome(), ndkComponent.InstallPathInAndroidHome())

	log.Printf("Done")

	log.Printf("Append NDK folder to $PATH")
	// Old NDK folder was deleted above, its path can stay in $PATH
	if err := tools.ExportEnvironmentWithEnvman("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), newNDKHome)); err != nil {
		return err
	}

	if err := tools.ExportEnvironmentWithEnvman(androidNDKHome, newNDKHome); err != nil {
		return err
	}
	log.Printf("Exported $%s: %s", androidNDKHome, newNDKHome)

	return nil
}
