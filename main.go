package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-core/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-steplib/steps-install-missing-android-tools/analyzer"
	"github.com/bitrise-steplib/steps-install-missing-android-tools/installer"
)

const (
	buildGradleBasename    = "build.gradle"
	settingsGradleBasename = "settings.gradle"
)

// ConfigsModel ...
type ConfigsModel struct {
	SourceDir                           string
	UpdateSupportLibraryAndPlayServices string
	AndroidHome                         string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		SourceDir:                           os.Getenv("source_dir"),
		UpdateSupportLibraryAndPlayServices: os.Getenv("update_support_library_and_play_services"),
		AndroidHome:                         os.Getenv("ANDROID_HOME"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")

	log.Printf("- SourceDir: %s", configs.SourceDir)
	log.Printf("- UpdateSupportLibraryAndPlayServices: %s", configs.UpdateSupportLibraryAndPlayServices)
	log.Printf("- AndroidHome: %s", configs.AndroidHome)
}

func (configs ConfigsModel) validate() error {
	if configs.SourceDir == "" {
		return errors.New("no SourceDir parameter specified")
	}
	if exist, err := pathutil.IsPathExists(configs.SourceDir); err != nil {
		return fmt.Errorf("failed to check if SourceDir exist at: %s, error: %s", configs.SourceDir, err)
	} else if !exist {
		return fmt.Errorf("sourceDir not exist at: %s", configs.SourceDir)
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
	// Search for root setting.gradle file
	fmt.Println()
	log.Infof("Search for root setting.gradle file")

	sourceDir, err := pathutil.AbsPath(configs.SourceDir)
	if err != nil {
		failf("Failed to expand path: %s", configs.SourceDir)
	}

	files, err := utility.ListPathInDirSortedByComponents(sourceDir, false)
	if err != nil {
		failf("Failed to list files at: %s", configs.SourceDir)
	}

	buildGradleFiles, err := filterBuildGradleFiles(files)
	if err != nil {
		failf("Failed to list build.gradle files, error: %s", err)
	}
	log.Printf("build.gradle files: %s", buildGradleFiles)

	if len(buildGradleFiles) == 0 {
		failf("No build.gradle file found")
	}

	rootBuildGradleFile := buildGradleFiles[0]
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

	if len(buildGradleFiles) == 1 {
		buildGradleFilesToAnalyze = append(buildGradleFilesToAnalyze, buildGradleFiles[0])
	} else {
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

		content, err := fileutil.ReadStringFromFile(buildGradleFile)
		if err != nil {
			failf("Failed to read build.gradle file at: %s, error: %s", buildGradleFile, err)
		}

		dependencies, err := analyzer.NewProjectDependencies(content)
		if err != nil {
			log.Errorf("Failed to parse build.gradle at: %s", buildGradleFile)
			continue
		}

		dependenciesToEnsure = append(dependenciesToEnsure, dependencies)
	}
	// ---

	//
	// Ensure dependencies
	fmt.Println()
	log.Infof("Ensure dependencies")

	toolHelper := installer.New(configs.AndroidHome)

	isSupportLibraryUpdated := false
	isGooglePlayServicesUpdated := false

	for _, dependencies := range dependenciesToEnsure {
		// Ensure SDK
		log.Printf("Checking compileSdkVersion: %s", dependencies.ComplieSDKVersion)

		if installed, err := toolHelper.IsSDKVersionInstalled(dependencies.ComplieSDKVersion); err != nil {
			failf("Failed to check if sdk version (%s) installed, error: %s", dependencies.ComplieSDKVersion.String(), err)
		} else if !installed {
			log.Printf("compileSdkVersion: %s not installed", dependencies.ComplieSDKVersion.String())

			if err := toolHelper.InstallSDKVersion(dependencies.ComplieSDKVersion); err != nil {
				failf("Failed to install sdk version (%s), error: %s", dependencies.ComplieSDKVersion.String(), err)
			}
		}

		log.Donef("compileSdkVersion: %s installed", dependencies.ComplieSDKVersion.String())

		// Ensure build-tool
		log.Printf("Checking buildToolsVersion: %s", dependencies.BuildToolsVersion)

		if installed, err := toolHelper.IsBuildToolsInstalled(dependencies.BuildToolsVersion); err != nil {
			failf("Failed to check if build-tools (%s) installed, error: %s", dependencies.BuildToolsVersion.String(), err)
		} else if !installed {
			log.Printf("buildToolsVersion: %s not installed", dependencies.BuildToolsVersion.String())

			if err := toolHelper.InstallBuildToolsVersion(dependencies.BuildToolsVersion); err != nil {
				failf("Failed to install build-tools version (%s), error: %s", dependencies.BuildToolsVersion.String(), err)
			}
		}

		log.Donef("buildToolsVersion: %s installed", dependencies.BuildToolsVersion.String())

		// Ensure support-library
		if dependencies.UseSupportLibrary && configs.UpdateSupportLibraryAndPlayServices == "true" && !isSupportLibraryUpdated {
			log.Printf("Updating Support Library")

			if err := toolHelper.UpdateSupportLibrary(); err != nil {
				failf("Failed to update Support Library, error: %s", err)
			}

			isSupportLibraryUpdated = true
			log.Donef("Support Library updated")
		}

		// Ensure google-play-services
		if dependencies.UseGooglePlayServices && configs.UpdateSupportLibraryAndPlayServices == "true" && !isGooglePlayServicesUpdated {
			log.Printf("Updating Google Play Services")

			if err := toolHelper.UpdateGooglePlayServices(); err != nil {
				failf("Failed to update Google Play Services, error: %s", err)
			}

			isGooglePlayServicesUpdated = true
			log.Donef("Google Play Services updated")
		}
	}
}
