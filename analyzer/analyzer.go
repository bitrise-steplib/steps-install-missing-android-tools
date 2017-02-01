package analyzer

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/hashicorp/go-version"
)

// -----------------------
// --- Stucts
// -----------------------

// ProjectDependenciesModel ...
type ProjectDependenciesModel struct {
	ComplieSDKVersion string
	BuildToolsVersion string

	UseSupportLibrary     bool
	UseGooglePlayServices bool
}

// NewProjectDependencies ...
func NewProjectDependencies(buildGradleContent, gradlewPath string) (ProjectDependenciesModel, error) {
	complieSDKVersion, buildToolsVersion, err := parseBuildGradle(buildGradleContent)
	if err != nil {
		return ProjectDependenciesModel{}, err
	}

	cmd := command.New(gradlewPath, "androidDependencies")
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return ProjectDependenciesModel{}, err
	}

	useSupportLibrary, useGooglePlayServices, err := parseAndroidDependencies(out)
	if err != nil {
		return ProjectDependenciesModel{}, err
	}

	return ProjectDependenciesModel{
		ComplieSDKVersion:     complieSDKVersion,
		BuildToolsVersion:     buildToolsVersion,
		UseSupportLibrary:     useSupportLibrary,
		UseGooglePlayServices: useGooglePlayServices,
	}, nil
}

func parseAndroidDependencies(out string) (bool, bool, error) {
	useSupportLibrary := false
	useGooglePlayServices := false

	supportLibrarayPattern := "com.android.support:support"
	googlePlayServicesPattern := "com.google.android.gms:play-services"

	reader := strings.NewReader(out)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, supportLibrarayPattern) {
			useSupportLibrary = true
			continue
		}
		if strings.Contains(line, googlePlayServicesPattern) {
			useGooglePlayServices = true
			continue
		}
		if useSupportLibrary && useGooglePlayServices {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return false, false, err
	}

	return useSupportLibrary, useGooglePlayServices, nil
}

// String ...
func (projectDepencies ProjectDependenciesModel) String() string {
	outStr := ""
	if projectDepencies.ComplieSDKVersion != "" {
		outStr += fmt.Sprintf("  compileSdkVersion: %s\n", projectDepencies.ComplieSDKVersion)
	}
	if projectDepencies.BuildToolsVersion != "" {
		outStr += fmt.Sprintf("  buildToolsVersion: %s\n", projectDepencies.BuildToolsVersion)
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

func parseCompileSDKVersion(buildGradleContent string) (string, error) {
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
		return "", err
	}

	if compileSDKVersionStr == "" {
		return "", errors.New("failed to find compileSdkVersion")
	}

	if _, err := version.NewVersion(compileSDKVersionStr); err != nil {
		// Possible defined with variable
		return "", fmt.Errorf("failed to parse compileSdkVersion (%s), error: %s", compileSDKVersionStr, err)
	}

	return compileSDKVersionStr, nil
}

func parseBuildToolsVersion(buildGradleContent string) (string, error) {
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
		return "", err
	}

	if buildToolsVersionStr == "" {
		return "", errors.New("failed to find buildToolsVersion")
	}

	if _, err := version.NewVersion(buildToolsVersionStr); err != nil {
		// Possible defined with variable
		return "", fmt.Errorf("failed to parse buildToolsVersion (%s), error: %s", buildToolsVersionStr, err)
	}

	return buildToolsVersionStr, nil
}

func parseBuildGradle(buildGradleContent string) (string, string, error) {
	compileSDKVersion, err := parseCompileSDKVersion(buildGradleContent)
	if err != nil {
		return "", "", err
	}

	buildToolsVersion, err := parseBuildToolsVersion(buildGradleContent)
	if err != nil {
		return "", "", err
	}

	return compileSDKVersion, buildToolsVersion, nil
}
