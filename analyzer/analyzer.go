package analyzer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/hashicorp/go-version"
)

// -----------------------
// --- Stucts
// -----------------------

// ProjectDependenciesModel ...
type ProjectDependenciesModel struct {
	PlatformVersion   string
	BuildToolsVersion string

	UseSupportLibrary     bool
	UseGooglePlayServices bool
}

// NewProjectDependencies ...
func NewProjectDependencies(buildGradlePth, gradlewPth string) (ProjectDependenciesModel, error) {
	buildGradleContent, err := fileutil.ReadStringFromFile(buildGradlePth)
	if err != nil {
		return ProjectDependenciesModel{}, err
	}

	dependencies, err := parseBuildGradle(buildGradleContent)
	if err != nil {
		log.Warnf("failed to parse build gradle file: %s, error: %s", buildGradlePth, err)
		log.Warnf("switch to extended analyzer...")

		compileSDKVersion, buildToolsVersion, err := analyzeWithGradlew(buildGradlePth, gradlewPth)
		if err != nil {
			return ProjectDependenciesModel{}, err
		}

		useSupportLibrary, err := parseUseSupportLibrary(buildGradleContent)
		if err != nil {
			log.Warnf("failed to detemin if use supportLibrary, error: %s", err)
		}

		useGooglePlayServices, err := parseUseGooglePlayServices(buildGradleContent)
		if err != nil {
			log.Warnf("failed to detemine if use googlePlayServices, error: %s", err)
		}

		return ProjectDependenciesModel{
			PlatformVersion:       compileSDKVersion,
			BuildToolsVersion:     buildToolsVersion,
			UseSupportLibrary:     useSupportLibrary,
			UseGooglePlayServices: useGooglePlayServices,
		}, nil
	}

	return dependencies, nil
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

func parsePlatformVersion(buildGradleContent string) (string, error) {
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

	_, err := version.NewVersion(compileSDKVersionStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse compileSdkVersion (%s), error: %s", compileSDKVersionStr, err)
	}

	return "android-" + compileSDKVersionStr, nil
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

	_, err := version.NewVersion(buildToolsVersionStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse buildToolsVersion (%s), error: %s", buildToolsVersionStr, err)
	}

	return buildToolsVersionStr, nil
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

func analyzeWithGradlew(buildGradlePth, gradlewPth string) (string, string, error) {
	analyzerScriptContent := `
if (plugins.hasPlugin('com.android.application')) {
	println 'compileSdkVersion: '+android.compileSdkVersion
	println 'buildToolsVersion: '+android.buildToolsVersion
}
`

	buildGradleContent, err := fileutil.ReadStringFromFile(buildGradlePth)
	if err != nil {
		return "", "", err
	}

	buildGradleContentWitnAnalyzerScript := buildGradleContent + analyzerScriptContent

	if err := fileutil.WriteStringToFile(buildGradlePth, buildGradleContentWitnAnalyzerScript); err != nil {
		return "", "", err
	}

	defer func() {
		if err := fileutil.WriteStringToFile(buildGradlePth, buildGradleContent); err != nil {
			log.Errorf("Failed to remove analyzer script from: %s, error: %s", buildGradlePth, err)
		}
	}()

	var outBuffer bytes.Buffer
	outWriter := io.Writer(&outBuffer)

	var errBuffer bytes.Buffer
	errWriter := io.Writer(&errBuffer)

	cmd := command.New(gradlewPth, "buildEnvironment")
	cmd.SetStdout(outWriter)
	cmd.SetStderr(errWriter)
	cmd.SetDir(filepath.Dir(buildGradlePth))

	if err := cmd.Run(); err != nil {

	}

	compileSDKVersion := ""
	buildToolsVersion := ""

	reader := strings.NewReader(outBuffer.String())
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "compileSdkVersion: ") {
			compileSDKVersion = strings.TrimPrefix(line, "compileSdkVersion: ")
		}

		if strings.HasPrefix(line, "buildToolsVersion: ") {
			buildToolsVersion = strings.TrimPrefix(line, "buildToolsVersion: ")
		}
	}

	if strings.HasPrefix(compileSDKVersion, "android-") {
		_, err = version.NewVersion(strings.TrimPrefix(compileSDKVersion, "android-"))
		if err != nil {
			return "", "", fmt.Errorf("failed to parse compileSDKVersion (%s), error: %s", compileSDKVersion, err)
		}
	} else {
		return "", "", fmt.Errorf("failed to parse compileSDKVersion (%s)", compileSDKVersion)
	}

	_, err = version.NewVersion(buildToolsVersion)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse buildToolsVersion (%s), error: %s", buildToolsVersion, err)
	}

	return compileSDKVersion, buildToolsVersion, nil
}

func parseBuildGradle(buildGradleContent string) (ProjectDependenciesModel, error) {
	platformVersion, err := parsePlatformVersion(buildGradleContent)
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
		PlatformVersion:   platformVersion,
		BuildToolsVersion: buildToolsVersion,

		UseSupportLibrary:     useSupportLibrary,
		UseGooglePlayServices: useGooglePlayServices,
	}

	return dependencies, nil
}
