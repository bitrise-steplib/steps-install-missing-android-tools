package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/install-missing-android-tools/analyzer"
	"github.com/bitrise-io/install-missing-android-tools/installer"
	log "github.com/bitrise-io/install-missing-android-tools/logger"
)

const (
	buildGradleBasename    = "build.gradle"
	settingsGradleBasename = "settings.gradle"
)

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

// ByComponents ..
type ByComponents []string

func (s ByComponents) Len() int {
	return len(s)
}
func (s ByComponents) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByComponents) Less(i, j int) bool {
	path1 := s[i]
	path2 := s[j]

	d1, err := pathDept(path1)
	if err != nil {
		log.Warn("failed to calculate path depth (%s), error: %s", path1, err)
		return false
	}

	d2, err := pathDept(path2)
	if err != nil {
		log.Warn("failed to calculate path depth (%s), error: %s", path1, err)
		return false
	}

	if d1 < d2 {
		return true
	} else if d1 > d2 {
		return false
	}

	// if same component size,
	// do alphabetic sort based on the last component
	base1 := filepath.Base(path1)
	base2 := filepath.Base(path2)

	return base1 < base2
}

// -----------------------
// --- Functions
// -----------------------

func validateRequiredInput(key, value string) {
	if value == "" {
		log.Fail("Missing required input: %s", key)
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

func filterBuildGradleFiles(fileList []string) []string {
	gradleFiles := filterFilesWithBasPaths(fileList, buildGradleBasename)

	if len(gradleFiles) > 0 {
		sort.Sort(ByComponents(gradleFiles))
	}

	return gradleFiles
}

func filterSettingsGradleFiles(fileList []string) []string {
	gradleFiles := filterFilesWithBasPaths(fileList, settingsGradleBasename)

	if len(gradleFiles) > 0 {
		sort.Sort(ByComponents(gradleFiles))
	}

	return gradleFiles
}

func filterRootBuildGradleFile(fileList []string) string {
	gradleFiles := filterFilesWithBasPaths(fileList, buildGradleBasename)
	if len(gradleFiles) == 0 {
		return ""
	}

	sort.Sort(ByComponents(gradleFiles))
	return gradleFiles[0]
}

// -----------------------
// --- Main
// -----------------------
func main() {
	// Input validation
	buildGradleFilesToAnalyze := []string{}

	sourceDir := os.Getenv("source_dir")
	updateSupportLibraryAndGooglePlayServices := os.Getenv("update_support_library_and_play_services")

	log.Configs(sourceDir, updateSupportLibraryAndGooglePlayServices)

	validateRequiredInput("source_dir", sourceDir)
	validateRequiredInput("update_support_library_and_play_services", updateSupportLibraryAndGooglePlayServices)

	// Analyze root settings.gradle file
	log.Info("Analyze root settings.gradle file")

	dir, err := pathutil.AbsPath(sourceDir)
	if err != nil {
		log.Fail("Failed to expand path: %s", sourceDir)
	}

	sourceDir = dir

	files, err := fileList(sourceDir)
	if err != nil {
		log.Fail("Failed to list files at: %s", sourceDir)
	}

	rootBuildGradleFile := filterRootBuildGradleFile(files)
	if rootBuildGradleFile == "" {
		log.Fail("No root build.gradle file foud in files: %v", files)
	}
	log.Details("root build.gradle file: %s", rootBuildGradleFile)

	rootBuildGradleDir := filepath.Dir(rootBuildGradleFile)

	rootSettingsGradleFile := filepath.Join(rootBuildGradleDir, settingsGradleBasename)

	if exist, err := pathutil.IsPathExists(rootSettingsGradleFile); err != nil {
		log.Fail("Failed to check if root settings.gradle exist at: %s, error: %s", rootSettingsGradleFile, err)
	} else if !exist {
		log.Fail("No root settings.gradle exist at: %s", rootSettingsGradleFile)
	}
	log.Details("root settings.gradle file:: %s", rootSettingsGradleFile)

	buildGradleFiles := filterBuildGradleFiles(files)

	if len(buildGradleFiles) == 0 {
		log.Fail("No build.gradle files found at: %s", sourceDir)
	}

	if len(buildGradleFiles) == 1 {
		buildGradleFilesToAnalyze = append(buildGradleFilesToAnalyze, buildGradleFiles[0])
	} else {
		rootSettingsGradleContent, err := fileutil.ReadStringFromFile(rootSettingsGradleFile)
		if err != nil {
			log.Fail("Failed to read settings.gradle at: %s, error: %s", rootSettingsGradleFile, rootSettingsGradleContent)
		}

		modules, err := analyzer.ParseIncludedModules(rootSettingsGradleContent)
		if err != nil {
			log.Fail("Failed to parse included modules from settings.gradle at: %s", rootSettingsGradleFile)
		}

		log.Details("active modules to analyze: %v", modules)

		for _, module := range modules {
			moduleBuildGradleFile := filepath.Join(rootBuildGradleDir, module, buildGradleBasename)

			if exist, err := pathutil.IsPathExists(moduleBuildGradleFile); err != nil {
				log.Fail("Failed to check if %s's build.gradle exist at: %s, error: %s", module, moduleBuildGradleFile, err)
			} else if !exist {
				log.Fail("build.gradle file not found for module: %s at: %s, error: %s", module, moduleBuildGradleFile, err)
			}

			buildGradleFilesToAnalyze = append(buildGradleFilesToAnalyze, moduleBuildGradleFile)
		}
	}
	log.Details("build.gradle files to analyze: %v", buildGradleFilesToAnalyze)

	for _, buildGradleFile := range buildGradleFilesToAnalyze {
		log.Info("Analyze build.gradle file: %s", buildGradleFile)

		content, err := fileutil.ReadStringFromFile(buildGradleFile)
		if err != nil {
			log.Fail("Failed to read build.gradle file at: %s", buildGradleFile)
		}

		containsCompileSDKVersion := true
		if !strings.Contains(content, "compileSdkVersion") {
			containsCompileSDKVersion = false
		}

		containsBuildToolsVersion := true
		if !strings.Contains(content, "buildToolsVersion") {
			containsBuildToolsVersion = false
		}

		if !containsCompileSDKVersion {
			log.Warn("build.gradle at: %s does not contains compileSdkVersion")
		}

		if !containsBuildToolsVersion {
			log.Warn("build.gradle at: %s does not contains buildToolsVersion")
		}

		if !containsCompileSDKVersion || !containsBuildToolsVersion {
			log.Warn("maybe not the project's module build.gradle provided")
		}

		// Ensure android dependencies
		dependencies, err := analyzer.NewProjectDependenciesModel(content)
		if err != nil {
			log.Fail("Failed to parse build.gradle at: %s", buildGradleFile)
		}

		fmt.Println(dependencies.String())

		toolHelper, err := installer.NewAndroidToolHelperModel()
		if err != nil {
			log.Fail("Failed to create tool helper, error: %s", err)
		}

		if installed, err := toolHelper.IsSDKVersionInstalled(dependencies.ComplieSDKVersion); err != nil {
			log.Fail("Failed to check if sdk version (%s) installed, error: %s", dependencies.ComplieSDKVersion.String(), err)
		} else if !installed {
			log.Details("compileSdkVersion: %s not installed", dependencies.ComplieSDKVersion.String())
			if err := toolHelper.InstallSDKVersion(dependencies.ComplieSDKVersion); err != nil {
				log.Fail("Failed to install sdk version (%s), error: %s", dependencies.ComplieSDKVersion.String(), err)
			}
		}
		log.Done("compileSdkVersion: %s installed", dependencies.ComplieSDKVersion.String())

		if installed, err := toolHelper.IsBuildToolsInstalled(dependencies.BuildToolsVersion); err != nil {
			log.Fail("Failed to check if build-tools (%s) installed, error: %s", dependencies.BuildToolsVersion.String(), err)
		} else if !installed {
			log.Details("buildToolsVersion: %s not installed", dependencies.BuildToolsVersion.String())
			if err := toolHelper.InstallBuildToolsVersion(dependencies.BuildToolsVersion); err != nil {
				log.Fail("Failed to install build-tools version (%s), error: %s", dependencies.BuildToolsVersion.String(), err)
			}
		}
		log.Done("buildToolsVersion: %s installed", dependencies.BuildToolsVersion.String())

		if dependencies.UseSupportLibrary {
			if updateSupportLibraryAndGooglePlayServices == "true" {
				log.Details("Updating Support Library")
				if err := toolHelper.UpdateSupportLibrary(); err != nil {
					log.Fail("Failed to update Support Library, error: %s", err)
				}
			} else {
				log.Warn("Project uses Support Library, but update_support_library_and_play_services is false")
			}
			log.Done("Support Library updated")
		}

		if dependencies.UseGooglePlayServices {
			if updateSupportLibraryAndGooglePlayServices == "true" {
				log.Details("Updating Google Play Services")
				if err := toolHelper.UpdateGooglePlayServices(); err != nil {
					log.Fail("Failed to update Google Play Services, error: %s", err)
				}
			} else {
				log.Warn("Project uses Google Play Services, but update_support_library_and_play_services is false")
			}
			log.Done("Google Play Services updated")
		}
	}
}
