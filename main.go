package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-core/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-install-missing-android-tools/analyzer"
	"github.com/bitrise-tools/go-android/sdk"
	"github.com/bitrise-tools/go-android/sdkcomponent"
	"github.com/bitrise-tools/go-android/sdkmanager"
)

const (
	buildGradleBasename    = "build.gradle"
	settingsGradleBasename = "settings.gradle"
)

// ConfigsModel ...
type ConfigsModel struct {
	RootBuildGradleFile                 string
	GradlewPath                         string
	UpdateSupportLibraryAndPlayServices string
	AndroidHome                         string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		RootBuildGradleFile:                 os.Getenv("root_build_gradle_file"),
		GradlewPath:                         os.Getenv("gradlew_path"),
		UpdateSupportLibraryAndPlayServices: os.Getenv("update_support_library_and_play_services"),
		AndroidHome:                         os.Getenv("ANDROID_HOME"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")

	log.Printf("- RootBuildGradleFile: %s", configs.RootBuildGradleFile)
	log.Printf("- GradlewPath: %s", configs.GradlewPath)
	log.Printf("- UpdateSupportLibraryAndPlayServices: %s", configs.UpdateSupportLibraryAndPlayServices)
	log.Printf("- AndroidHome: %s", configs.AndroidHome)
}

func (configs ConfigsModel) validate() error {
	if configs.RootBuildGradleFile == "" {
		return errors.New("no RootBuildGradleFile parameter specified")
	}
	if exist, err := pathutil.IsPathExists(configs.RootBuildGradleFile); err != nil {
		return fmt.Errorf("failed to check if RootBuildGradleFile exist at: %s, error: %s", configs.RootBuildGradleFile, err)
	} else if !exist {
		return fmt.Errorf("RootBuildGradleFile not exist at: %s", configs.RootBuildGradleFile)
	}

	if configs.GradlewPath == "" {
		return errors.New("no GradlewPath parameter specified")
	}
	if exist, err := pathutil.IsPathExists(configs.GradlewPath); err != nil {
		return fmt.Errorf("failed to check if GradlewPath exist at: %s, error: %s", configs.GradlewPath, err)
	} else if !exist {
		return fmt.Errorf("GradlewPath not exist at: %s", configs.GradlewPath)
	}

	if configs.UpdateSupportLibraryAndPlayServices == "" {
		return errors.New("no UpdateSupportLibraryAndPlayServices parameter specified")
	}
	if configs.UpdateSupportLibraryAndPlayServices != "true" && configs.UpdateSupportLibraryAndPlayServices != "false" {
		return fmt.Errorf("invalid UpdateSupportLibraryAndPlayServices provided: %s, vaialable: [true false]", configs.UpdateSupportLibraryAndPlayServices)
	}

	if configs.AndroidHome == "" {
		return fmt.Errorf("no ANDROID_HOME set")
	}

	return nil
}

// -----------------------
// --- Functions
// -----------------------

func filterBuildGradleFiles(fileList []string) ([]string, error) {
	allowBuildGradleBaseFilter := utility.BaseFilter(buildGradleBasename, true)
	gradleFiles, err := utility.FilterPaths(fileList, allowBuildGradleBaseFilter)
	if err != nil {
		return []string{}, err
	}

	return utility.SortPathsByComponents(gradleFiles)
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
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
		failf("Issue with input: %s", err)
	}

	//
	// Search for root settings.gradle file
	fmt.Println()
	log.Infof("Search for root settings.gradle file")

	rootBuildGradleFile, err := pathutil.AbsPath(configs.RootBuildGradleFile)
	if err != nil {
		failf("Failed to expand root build.gradle file path (%s), error: %s", configs.RootBuildGradleFile, err)
	}

	log.Printf("root build.gradle file: %s", rootBuildGradleFile)

	// root settigs.gradle file should be in the same dir as root build.gradle file
	rootBuildGradleDir := filepath.Dir(rootBuildGradleFile)
	rootSettingsGradleFile := filepath.Join(rootBuildGradleDir, settingsGradleBasename)
	if exist, err := pathutil.IsPathExists(rootSettingsGradleFile); err != nil {
		failf("Failed to check if root settings.gradle exist at: %s, error: %s", rootSettingsGradleFile, err)
	} else if !exist {
		failf("No root settings.gradle exist at: %s", rootSettingsGradleFile)
	}

	log.Printf("root settings.gradle file:: %s", rootSettingsGradleFile)
	// ---

	//
	// Collect build.gradle files to analyze based on root settings.gradle file
	fmt.Println()
	log.Infof("Collect build.gradle files to analyze")

	buildGradleFilesToAnalyze := []string{}

	rootSettingsGradleContent, err := fileutil.ReadStringFromFile(rootSettingsGradleFile)
	if err != nil {
		failf("Failed to read settings.gradle at: %s, error: %s", rootSettingsGradleFile, err)
	}

	modules, err := analyzer.ParseIncludedModules(rootSettingsGradleContent)
	if err != nil {
		failf("Failed to parse included modules from settings.gradle at: %s, error: %s", rootSettingsGradleFile, err)

	}

	log.Printf("active modules to analyze: %v", modules)

	for _, module := range modules {
		moduleBuildGradleFile := filepath.Join(rootBuildGradleDir, module, buildGradleBasename)
		if exist, err := pathutil.IsPathExists(moduleBuildGradleFile); err != nil {
			failf("Failed to check if %s's build.gradle exist at: %s, error: %s", module, moduleBuildGradleFile, err)
		} else if !exist {
			log.Warnf("build.gradle file not found for module: %s at: %s, error: %s", module, moduleBuildGradleFile, err)
			continue
		}

		buildGradleFilesToAnalyze = append(buildGradleFilesToAnalyze, moduleBuildGradleFile)
	}

	log.Printf("build.gradle files to analyze: %v", buildGradleFilesToAnalyze)
	// ---

	//
	// Collect dependencies to ensure
	fmt.Println()
	log.Infof("Collect dependencies to ensure")

	dependenciesToEnsure := []analyzer.ProjectDependenciesModel{}

	for _, buildGradleFile := range buildGradleFilesToAnalyze {
		log.Printf("Analyze build.gradle file: %s", buildGradleFile)

		dependencies, err := analyzer.NewProjectDependencies(buildGradleFile, configs.GradlewPath)
		if err != nil {
			log.Errorf("Failed to analyze build.gradle at: %s, error: %s", buildGradleFile, err)
			continue
		}

		dependenciesToEnsure = append(dependenciesToEnsure, dependencies)
	}
	// ---

	//
	// Ensure dependencies
	fmt.Println()
	log.Infof("Ensure dependencies")

	androidSdk, err := sdk.New(configs.AndroidHome)
	if err != nil {
		failf("Failed to create sdk, error: %s", err)
	}

	sdkManager, err := sdkmanager.New(androidSdk)
	if err != nil {
		failf("Failed to create sdk manager, error: %s", err)
	}

	isSupportLibraryUpdated := false
	isGooglePlayServicesUpdated := false

	for _, dependencies := range dependenciesToEnsure {
		// Ensure SDK
		log.Printf("Checking compileSdkVersion: %s", dependencies.PlatformVersion)

		platformComponent := sdkcomponent.Platform{
			Version: dependencies.PlatformVersion,
		}

		if installed, err := sdkManager.IsInstalled(platformComponent); err != nil {
			failf("Failed to check if sdk version (%s) installed, error: %s", dependencies.PlatformVersion, err)
		} else if !installed {
			log.Printf("compileSdkVersion: %s not installed, installing...", dependencies.PlatformVersion)

			cmd := sdkManager.InstallCommand(platformComponent)
			cmd.SetStdin(strings.NewReader("y"))
			cmd.SetStdout(os.Stdout)
			cmd.SetStderr(os.Stderr)

			log.Printf("$ %s", cmd.PrintableCommandArgs())

			if err := cmd.Run(); err != nil {
				failf("Failed to install sdk version (%s), error: %s", dependencies.PlatformVersion, err)
			}
		}

		log.Donef("compileSdkVersion: %s installed", dependencies.PlatformVersion)

		// Ensure build-tools
		log.Printf("Checking buildToolsVersion: %s", dependencies.BuildToolsVersion)

		buildToolComponent := sdkcomponent.BuildTool{
			Version: dependencies.BuildToolsVersion,
		}

		if installed, err := sdkManager.IsInstalled(buildToolComponent); err != nil {
			failf("Failed to check if build-tools (%s) installed, error: %s", dependencies.BuildToolsVersion, err)
		} else if !installed {
			log.Printf("buildToolsVersion: %s not installed, installing...", dependencies.BuildToolsVersion)

			cmd := sdkManager.InstallCommand(buildToolComponent)
			cmd.SetStdin(strings.NewReader("y"))
			cmd.SetStdout(os.Stdout)
			cmd.SetStderr(os.Stderr)

			log.Printf("$ %s", cmd.PrintableCommandArgs())

			if err := cmd.Run(); err != nil {
				failf("Failed to install build tools version (%s), error: %s", dependencies.BuildToolsVersion, err)
			}

			if installed, err := sdkManager.IsInstalled(buildToolComponent); err != nil {
				failf("Failed to check if build tools version (%s) installed, error: %s", dependencies.BuildToolsVersion, err)
			} else if !installed {
				failf("Failed to install build tools version (%s)", dependencies.BuildToolsVersion)
			}

		}

		log.Donef("buildToolsVersion: %s installed", dependencies.BuildToolsVersion)

		// Ensure support-library
		if dependencies.UseSupportLibrary && configs.UpdateSupportLibraryAndPlayServices == "true" && !isSupportLibraryUpdated {
			log.Printf("Updating Support Library")

			extras := []sdkcomponent.Extras{}
			if sdkManager.IsLegacySDK() {
				extras = sdkcomponent.LegacySupportLibraryInstallComponents()
			} else {
				extras = sdkcomponent.SupportLibraryInstallComponents()
			}

			for _, extra := range extras {
				cmd := sdkManager.InstallCommand(extra)
				cmd.SetStdin(strings.NewReader("y"))
				cmd.SetStdout(os.Stdout)
				cmd.SetStderr(os.Stderr)

				log.Printf("$ %s", cmd.PrintableCommandArgs())

				if err := cmd.Run(); err != nil {
					failf("Failed to update Support Library, error: %s", err)
				}

				if installed, err := sdkManager.IsInstalled(extra); err != nil {
					failf("Failed to check if Support Library installed, error: %s", err)
				} else if !installed {
					failf("Failed to update Support Library, error: %s", err)
				}
			}

			isSupportLibraryUpdated = true
			log.Donef("Support Library updated")
		}

		// Ensure google-play-services
		if dependencies.UseGooglePlayServices && configs.UpdateSupportLibraryAndPlayServices == "true" && !isGooglePlayServicesUpdated {
			log.Printf("Updating Google Play Services")

			extras := []sdkcomponent.Extras{}
			if sdkManager.IsLegacySDK() {
				extras = sdkcomponent.LegacyGooglePlayServicesInstallComponents()
			} else {
				extras = sdkcomponent.GooglePlayServicesInstallComponents()
			}

			for _, extra := range extras {
				cmd := sdkManager.InstallCommand(extra)
				cmd.SetStdin(strings.NewReader("y"))
				cmd.SetStdout(os.Stdout)
				cmd.SetStderr(os.Stderr)

				log.Printf("$ %s", cmd.PrintableCommandArgs())

				if err := cmd.Run(); err != nil {
					failf("Failed to update Google Play Services, error: %s", err)
				}

				if installed, err := sdkManager.IsInstalled(extra); err != nil {
					failf("Failed to check if Google Play Services installed, error: %s", err)
				} else if !installed {
					failf("Failed to update Google Play Services, error: %s", err)
				}
			}

			isGooglePlayServicesUpdated = true
			log.Donef("Google Play Services updated")
		}
	}
}
