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

// NewProjectDependencies ...
func NewProjectDependencies(buildGradleContent string) (ProjectDependenciesModel, error) {
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
				module = strings.Trim(module, `'`)
				module = strings.Trim(module, `"`)
				module = strings.TrimPrefix(module, ":")

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

	pattern := `(?i).*compileSdkVersion\s*(?P<v>[0-9]+)`
	re := regexp.MustCompile(pattern)

	compileSDKVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			compileSDKVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if compileSDKVersionStr == "" {
		return nil, errors.New("failed to find compileSdkVersion")
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

	pattern := `(?i).*buildToolsVersion\s*["']+(?P<v>[0-9.]+)["']+`
	re := regexp.MustCompile(pattern)

	buildToolsVersionStr := ""

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			buildToolsVersionStr = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if buildToolsVersionStr == "" {
		return nil, errors.New("failed to find buildToolsVersion")
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
	//     androidTestCompile('com.android.support.test.espresso:espresso-core:2.2.2', {

	pattern := `(?i).*compile.*['"]+com.android.support.*['"]+`
	re := regexp.MustCompile(pattern)

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		line := scanner.Text()
		if match := re.FindString(line); match != "" {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func parseUseGooglePlayServices(buildGradleContent string) (bool, error) {
	//     compile "com.google.android.gms:play-services-location:7.8.0"

	pattern := `(?i).*compile.*['"]+com.google.android.gms.*play-services.*['"]+`
	re := regexp.MustCompile(pattern)

	scanner := bufio.NewScanner(strings.NewReader(buildGradleContent))
	for scanner.Scan() {
		line := scanner.Text()
		if match := re.FindString(line); match != "" {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func parseBuildGradle(buildGradleContent string) (ProjectDependenciesModel, error) {
	compileSDKVersion, err := parseCompileSDKVersion(buildGradleContent)
	if err != nil {
		return ProjectDependenciesModel{}, fmt.Errorf("failed to determine compileSDKVersion, error: %s", err)
	}

	buildToolsVersion, err := parseBuildToolsVersion(buildGradleContent)
	if err != nil {
		return ProjectDependenciesModel{}, fmt.Errorf("failed to deterime buildToolsVersion, error: %s", err)
	}

	useSupportLibrary, err := parseUseSupportLibrary(buildGradleContent)
	if err != nil {
		return ProjectDependenciesModel{}, fmt.Errorf("failed to detemin if use supportLibrary, error: %s", err)
	}

	useGooglePlayServices, err := parseUseGooglePlayServices(buildGradleContent)
	if err != nil {
		return ProjectDependenciesModel{}, fmt.Errorf("failed to detemine if use googlePlayServices, error: %s", err)
	}

	dependencies := ProjectDependenciesModel{
		ComplieSDKVersion: compileSDKVersion,
		BuildToolsVersion: buildToolsVersion,

		UseSupportLibrary:     useSupportLibrary,
		UseGooglePlayServices: useGooglePlayServices,
	}

	return dependencies, nil
}
