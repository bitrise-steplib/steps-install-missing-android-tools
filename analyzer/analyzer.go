package analyzer

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"
)

// -----------------------
// --- Stucts
// -----------------------

// ProjectDependenciesModel ...
type ProjectDependenciesModel struct {
	ComplieSDKVersion *version.Version
	BuildToolsVersion *version.Version

	UseSupportLibrary     bool
	UseGooglePlayServices bool
}

// NewProjectDependenciesModel ...
func NewProjectDependenciesModel(buildGradleContent string) (ProjectDependenciesModel, error) {
	return parseBuildGradle(buildGradleContent)
}

// String ...
func (projectDepencies ProjectDependenciesModel) String() string {
	outStr := ""
	if projectDepencies.ComplieSDKVersion != nil {
		outStr += fmt.Sprintf("  compileSdkVersion: %s\n", projectDepencies.ComplieSDKVersion.String())
	}
	if projectDepencies.BuildToolsVersion != nil {
		outStr += fmt.Sprintf("  buildToolsVersion: %s\n", projectDepencies.BuildToolsVersion.String())
	}

	outStr += fmt.Sprintf("  uses Support Library: %v\n", projectDepencies.UseSupportLibrary)
	outStr += fmt.Sprintf("  uses Google Play Services: %v\n", projectDepencies.UseGooglePlayServices)
	return outStr
}

// ParseIncludedModules ...
func ParseIncludedModules(settingsGradleContent string) ([]string, error) {
	// include ':app', ':dynamicgrid'
	includeRegexp := regexp.MustCompile(`\s*include\s*(?P<modules>.*)`)
	modules := []string{}

	scanner := bufio.NewScanner(strings.NewReader(settingsGradleContent))
	for scanner.Scan() {
		matches := includeRegexp.FindStringSubmatch(scanner.Text())

		if len(matches) > 1 {
			includeStr := matches[1]
			splits := strings.Split(includeStr, ",")
			for _, split := range splits {
				module := strings.TrimSpace(split)

				if strings.HasPrefix(module, `'`) {
					module = strings.Trim(module, `'`)
				} else if strings.HasPrefix(module, `"`) {
					module = strings.Trim(module, `"`)
				}

				if strings.HasPrefix(module, ":") {
					module = strings.TrimPrefix(module, ":")
				}

				modules = append(modules, module)
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return []string{}, err
	}

	return modules, nil
}

// -----------------------
// --- Functions
// -----------------------

func parseCompileSDKVersion(buildGradleContent string) (*version.Version, error) {
	//     compileSdkVersion 23
	compileSDKVersionRegexp := regexp.MustCompile(`\s*compileSdkVersion (?P<version>.+)`)
	compileSDKVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		matches := compileSDKVersionRegexp.FindStringSubmatch(scanner.Text())
		if len(matches) > 1 {
			compileSDKVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if compileSDKVersionStr == "" {
		return nil, errors.New("Failed to find compileSdkVersion")
	}

	compileSDKVesrion, err := version.NewVersion(compileSDKVersionStr)
	if err != nil {
		// Possible defined with variable
		return nil, fmt.Errorf("failed to parse compileSdkVersion (%s), error: %s", compileSDKVersionStr, err)
	}

	return compileSDKVesrion, nil
}

func parseBuildToolsVersion(buildGradleContent string) (*version.Version, error) {
	//     buildToolsVersion "23.0.3"
	buildToolsVersionRegexp := regexp.MustCompile(`\s*buildToolsVersion "(?P<version>.+)"`)
	buildToolsVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		matches := buildToolsVersionRegexp.FindStringSubmatch(scanner.Text())
		if len(matches) > 1 {
			buildToolsVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if buildToolsVersionStr == "" {
		return nil, errors.New("Failed to find buildToolsVersion")
	}

	buildToolsVersion, err := version.NewVersion(buildToolsVersionStr)
	if err != nil {
		// Possible defined with variable
		return nil, fmt.Errorf("failed to parse buildToolsVersion (%s), error: %s", buildToolsVersionStr, err)
	}

	return buildToolsVersion, nil
}

func parseUseSupportLibrary(buildGradleContent string) (bool, error) {
	//     compile "com.android.support:appcompat-v7:23.4.0"
	//     compile "com.android.support:23.4.0"
	supportLibraryVersionRegexp := regexp.MustCompile(`\s*compile\s*\"com.android.support:(?P<tool>.[^:]*):*(?P<version>.*)\"`)
	supportLibraryVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		matches := supportLibraryVersionRegexp.FindStringSubmatch(scanner.Text())
		if len(matches) > 2 {
			supportLibraryVersionStr = matches[2]
			break
		} else if len(matches) > 1 {
			supportLibraryVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return (supportLibraryVersionStr != ""), nil
}

func parseUseGooglePlayServices(buildGradleContent string) (bool, error) {
	//     compile "com.google.android.gms:play-services-location:7.8.0"
	//     compile "com.google.android.gms:7.8.0"
	googlePlayServicesVersionRegexp := regexp.MustCompile(`\s*compile\s*\"com.google.android.gms:(?P<tool>.[^:]*):*(?P<version>.*)\"`)
	googlePlayServicesVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		matches := googlePlayServicesVersionRegexp.FindStringSubmatch(scanner.Text())
		if len(matches) > 2 {
			googlePlayServicesVersionStr = matches[2]
			break
		} else if len(matches) > 1 {
			googlePlayServicesVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return (googlePlayServicesVersionStr != ""), nil
}

func parseBuildGradle(buildGradleContent string) (ProjectDependenciesModel, error) {
	compileSDKVersion, err := parseCompileSDKVersion(buildGradleContent)
	if err != nil {
		return ProjectDependenciesModel{}, err
	}

	buildToolsVersion, err := parseBuildToolsVersion(buildGradleContent)
	if err != nil {
		return ProjectDependenciesModel{}, err
	}

	useSupportLibrary, err := parseUseSupportLibrary(buildGradleContent)
	if err != nil {
		return ProjectDependenciesModel{}, err
	}

	useGooglePlayServices, err := parseUseGooglePlayServices(buildGradleContent)
	if err != nil {
		return ProjectDependenciesModel{}, err
	}

	dependencies := ProjectDependenciesModel{
		ComplieSDKVersion: compileSDKVersion,
		BuildToolsVersion: buildToolsVersion,

		UseSupportLibrary:     useSupportLibrary,
		UseGooglePlayServices: useGooglePlayServices,
	}

	return dependencies, nil
}
