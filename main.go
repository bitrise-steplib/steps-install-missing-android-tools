package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		SourceDir:                           os.Getenv("source_dir"),
		UpdateSupportLibraryAndPlayServices: os.Getenv("update_support_library_and_play_services"),
	}
}

func (configs ConfigsModel) print() {
	log.Info("Configs:")

	log.Detail("- SourceDir: %s", configs.SourceDir)
	log.Detail("- UpdateSupportLibraryAndPlayServices: %s", configs.UpdateSupportLibraryAndPlayServices)
}

func (configs ConfigsModel) validate() error {
	if configs.SourceDir == "" {
		return errors.New("No SourceDir parameter specified!")
	}
	if exist, err := pathutil.IsPathExists(configs.SourceDir); err != nil {
		return fmt.Errorf("Failed to check if SourceDir exist at: %s, error: %s", configs.SourceDir, err)
	} else if !exist {
		return fmt.Errorf("SourceDir not exist at: %s", configs.SourceDir)
	}

	if configs.UpdateSupportLibraryAndPlayServices == "" {
		return errors.New("No UpdateSupportLibraryAndPlayServices parameter specified!")
	}

	return nil
}

// -----------------------
// --- Sorting
// -----------------------

// PathDept ...
func pathDept(pth string) (int, error) {
	abs, err := pathutil.AbsPath(pth)
	if err != nil {
		return 0, err
	}
	comp := strings.Split(abs, string(os.PathSeparator))

	fixedComp := []string{}
	for _, c := range comp {
		if c != "" {
			fixedComp = append(fixedComp, c)
		}
	}

	return len(fixedComp), nil
}

// SortableAbsPath ...
type SortableAbsPath struct {
	pth           string
	pthComponents []string
}

// NewSortableAbsPath ...
func NewSortableAbsPath(pth string) (SortableAbsPath, error) {
	absPth, err := pathutil.AbsPath(pth)
	if err != nil {
		return SortableAbsPath{}, err
	}

	components := strings.Split(absPth, string(os.PathSeparator))

	fixedComponents := []string{}
	for _, c := range components {
		if c != "" {
			fixedComponents = append(fixedComponents, c)
		}
	}

	return SortableAbsPath{
		pth:           absPth,
		pthComponents: fixedComponents,
	}, nil
}

// ByComponents ..
type ByComponents []SortableAbsPath

func (s ByComponents) Len() int {
	return len(s)
}
func (s ByComponents) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByComponents) Less(i, j int) bool {
	sortablePath1 := s[i]
	sortablePath2 := s[j]

	depth1 := len(sortablePath1.pthComponents)
	depth2 := len(sortablePath2.pthComponents)

	if depth1 < depth2 {
		return true
	} else if depth1 > depth2 {
		return false
	}

	// if same component size,
	// do alphabetical sort based on the last component
	base1 := filepath.Base(sortablePath1.pth)
	base2 := filepath.Base(sortablePath2.pth)

	if base1 < base2 {
		return true
	}

	return false
}

// -----------------------
// --- Functions
// -----------------------

func validateRequiredInput(key, value string) {
	if value == "" {
		log.Error("Missing required input: %s", key)
		os.Exit(1)
	}
}

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	envman := exec.Command("envman", "add", "--key", keyStr)
	envman.Stdin = strings.NewReader(valueStr)
	envman.Stdout = os.Stdout
	envman.Stderr = os.Stderr
	return envman.Run()
}

func fileList(searchDir string) ([]string, error) {
	searchDir, err := filepath.Abs(searchDir)
	if err != nil {
		return []string{}, err
	}

	fileList := []string{}

	if err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)

		return nil
	}); err != nil {
		return []string{}, err
	}
	return fileList, nil
}

func filterFilesWithBasPaths(fileList []string, basePath ...string) []string {
	filteredFileList := []string{}

	for _, file := range fileList {
		base := filepath.Base(file)

		for _, desiredBasePath := range basePath {
			if strings.EqualFold(base, desiredBasePath) {
				filteredFileList = append(filteredFileList, file)
				break
			}
		}
	}

	return filteredFileList
}

func filterBuildGradleFiles(fileList []string) ([]string, error) {
	gradleFiles := filterFilesWithBasPaths(fileList, buildGradleBasename)

	if len(gradleFiles) > 0 {
		sortabelPths := []SortableAbsPath{}
		for _, pth := range gradleFiles {
			sortablePth, err := NewSortableAbsPath(pth)
			if err != nil {
				return []string{}, err
			}
			sortabelPths = append(sortabelPths, sortablePth)
		}
		sort.Sort(ByComponents(sortabelPths))

		sortedPths := []string{}
		for _, sortablePth := range sortabelPths {
			sortedPths = append(sortedPths, sortablePth.pth)
		}

		gradleFiles = sortedPths
	}

	return gradleFiles, nil
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
		log.Error("Issue with input: %s", err)
		if err := exportEnvironmentWithEnvman("BITRISE_XAMARIN_TEST_RESULT", "failed"); err != nil {
			log.Warn("Failed to export environment: %s, error: %s", "BITRISE_XAMARIN_TEST_RESULT", err)
		}
		os.Exit(1)
	}
	fmt.Println()

	//
	// Search for root setting.gradle file
	log.Info("Analyze root settings.gradle file")

	dir, err := pathutil.AbsPath(configs.SourceDir)
	if err != nil {
		log.Error("Failed to expand path: %s", configs.SourceDir)
		os.Exit(1)
	}

	configs.SourceDir = dir

	files, err := fileList(configs.SourceDir)
	if err != nil {
		log.Error("Failed to list files at: %s", configs.SourceDir)
		os.Exit(1)
	}

	buildGradleFiles, err := filterBuildGradleFiles(files)
	if err != nil {
		log.Error("Failed to list build.gradle files, error: %s", err)
		os.Exit(1)
	}

	if len(buildGradleFiles) == 0 {
		log.Error("No build.gradle file found")
		os.Exit(1)
	}

	rootBuildGradleFile := buildGradleFiles[0]
	log.Detail("root build.gradle file: %s", rootBuildGradleFile)

	// root settigs.gradle file should be in the same dir as root build.gradle file
	rootBuildGradleDir := filepath.Dir(rootBuildGradleFile)
	rootSettingsGradleFile := filepath.Join(rootBuildGradleDir, settingsGradleBasename)
	if exist, err := pathutil.IsPathExists(rootSettingsGradleFile); err != nil {
		log.Error("Failed to check if root settings.gradle exist at: %s, error: %s", rootSettingsGradleFile, err)
		os.Exit(1)
	} else if !exist {
		log.Error("No root settings.gradle exist at: %s", rootSettingsGradleFile)
		os.Exit(1)
	}

	log.Detail("root settings.gradle file:: %s", rootSettingsGradleFile)
	// ---

	//
	// Collect build.gradle files to analyze based on root settings.gradle file
	buildGradleFilesToAnalyze := []string{}

	if len(buildGradleFiles) == 1 {
		buildGradleFilesToAnalyze = append(buildGradleFilesToAnalyze, buildGradleFiles[0])
	} else {
		rootSettingsGradleContent, err := fileutil.ReadStringFromFile(rootSettingsGradleFile)
		if err != nil {
			log.Error("Failed to read settings.gradle at: %s, error: %s", rootSettingsGradleFile, err)
			os.Exit(1)
		}

		modules, err := analyzer.ParseIncludedModules(rootSettingsGradleContent)
		if err != nil {
			log.Error("Failed to parse included modules from settings.gradle at: %s, error: %s", rootSettingsGradleFile, err)
			os.Exit(1)
		}

		log.Detail("active modules to analyze: %v", modules)

		for _, module := range modules {
			moduleBuildGradleFile := filepath.Join(rootBuildGradleDir, module, buildGradleBasename)
			if exist, err := pathutil.IsPathExists(moduleBuildGradleFile); err != nil {
				log.Error("Failed to check if %s's build.gradle exist at: %s, error: %s", module, moduleBuildGradleFile, err)
				os.Exit(1)
			} else if !exist {
				log.Warn("build.gradle file not found for module: %s at: %s, error: %s", module, moduleBuildGradleFile, err)
				continue
			}

			buildGradleFilesToAnalyze = append(buildGradleFilesToAnalyze, moduleBuildGradleFile)
		}
	}

	log.Detail("build.gradle files to analyze: %v", buildGradleFilesToAnalyze)
	// ---

	//
	// Collect dependencies to ensure
	dependenciesToEnsure := []analyzer.ProjectDependenciesModel{}

	for _, buildGradleFile := range buildGradleFilesToAnalyze {
		log.Info("Analyze build.gradle file: %s", buildGradleFile)

		content, err := fileutil.ReadStringFromFile(buildGradleFile)
		if err != nil {
			log.Error("Failed to read build.gradle file at: %s, error: %s", buildGradleFile, err)
			os.Exit(1)
		}

		dependencies, err := analyzer.NewProjectDependenciesModel(content)
		if err != nil {
			log.Error("Failed to parse build.gradle at: %s", buildGradleFile)
			os.Exit(1)
		}

		dependenciesToEnsure = append(dependenciesToEnsure, dependencies)

	}
	// ---

	//
	// Ensure dependencies
	toolHelper, err := installer.NewAndroidToolHelperModel()
	if err != nil {
		log.Error("Failed to create tool helper, error: %s", err)
		os.Exit(1)
	}

	isSupportLibraryUpdated := false
	isGooglePlayServicesUpdated := false

	for _, dependencies := range dependenciesToEnsure {

		// Ensure SDK
		log.Detail("Checking compileSdkVersion: %s", dependencies.ComplieSDKVersion)

		if installed, err := toolHelper.IsSDKVersionInstalled(dependencies.ComplieSDKVersion); err != nil {
			log.Error("Failed to check if sdk version (%s) installed, error: %s", dependencies.ComplieSDKVersion.String(), err)
			os.Exit(1)
		} else if !installed {
			log.Detail("compileSdkVersion: %s not installed", dependencies.ComplieSDKVersion.String())

			if err := toolHelper.InstallSDKVersion(dependencies.ComplieSDKVersion); err != nil {
				log.Error("Failed to install sdk version (%s), error: %s", dependencies.ComplieSDKVersion.String(), err)
				os.Exit(1)
			}
		}

		log.Done("compileSdkVersion: %s installed", dependencies.ComplieSDKVersion.String())

		// Ensure build-tool
		log.Detail("Checking buildToolsVersion: %s", dependencies.BuildToolsVersion)

		if installed, err := toolHelper.IsBuildToolsInstalled(dependencies.BuildToolsVersion); err != nil {
			log.Error("Failed to check if build-tools (%s) installed, error: %s", dependencies.BuildToolsVersion.String(), err)
			os.Exit(1)
		} else if !installed {
			log.Detail("buildToolsVersion: %s not installed", dependencies.BuildToolsVersion.String())

			if err := toolHelper.InstallBuildToolsVersion(dependencies.BuildToolsVersion); err != nil {
				log.Error("Failed to install build-tools version (%s), error: %s", dependencies.BuildToolsVersion.String(), err)
				os.Exit(1)
			}
		}

		log.Done("buildToolsVersion: %s installed", dependencies.BuildToolsVersion.String())

		// Ensure support-library
		if dependencies.UseSupportLibrary && configs.UpdateSupportLibraryAndPlayServices == "true" && !isSupportLibraryUpdated {
			log.Detail("Updating Support Library")

			if err := toolHelper.UpdateSupportLibrary(); err != nil {
				log.Error("Failed to update Support Library, error: %s", err)
				os.Exit(1)
			}

			isSupportLibraryUpdated = true
			log.Done("Support Library updated")
		}

		// Ensure google-play-services
		if dependencies.UseGooglePlayServices && configs.UpdateSupportLibraryAndPlayServices == "true" && !isGooglePlayServicesUpdated {
			log.Detail("Updating Google Play Services")

			if err := toolHelper.UpdateGooglePlayServices(); err != nil {
				log.Error("Failed to update Google Play Services, error: %s", err)
				os.Exit(1)
			}

			isGooglePlayServicesUpdated = true
			log.Done("Google Play Services updated")
		}
	}
}
