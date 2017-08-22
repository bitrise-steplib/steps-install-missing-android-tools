package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-android/sdk"
	"github.com/bitrise-tools/go-android/sdkcomponent"
	"github.com/bitrise-tools/go-android/sdkmanager"
	version "github.com/hashicorp/go-version"
)

// ConfigsModel ...
type ConfigsModel struct {
	RootBuildGradleFile string
	GradlewPath         string
	AndroidHome         string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		RootBuildGradleFile: os.Getenv("root_build_gradle_file"),
		GradlewPath:         os.Getenv("gradlew_path"),
		AndroidHome:         os.Getenv("ANDROID_HOME"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")

	log.Printf("- RootBuildGradleFile: %s", configs.RootBuildGradleFile)
	log.Printf("- GradlewPath: %s", configs.GradlewPath)
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

	if configs.AndroidHome == "" {
		return fmt.Errorf("no ANDROID_HOME set")
	}

	return nil
}

// -----------------------
// --- Functions
// -----------------------

// AutoDownloadSDKComponentsMinGradlePluginVersion ...
var AutoDownloadSDKComponentsMinGradlePluginVersion = version.Must(version.NewVersion("2.2.0"))

func androidGradlePluginVersionFromContent(rootBuildGradleContent string) (*version.Version, error) {
	reader := strings.NewReader(rootBuildGradleContent)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// classpath 'com.android.tools.build:gradle:2.1.3'
		pattern := `\s*classpath\s*'com.android.tools.build:gradle:(?P<version>[0-9.*]*)'\s*`
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); len(matches) == 2 {
			pluginVersionStr := matches[1]
			pluginVersion, err := version.NewVersion(pluginVersionStr)
			if err != nil {
				return nil, err
			}
			return pluginVersion, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("missing android gradle plugin version")
}

// AndroidGradlePluginVersion ...
func AndroidGradlePluginVersion(rootBuildGradlePth string) (*version.Version, error) {
	content, err := fileutil.ReadStringFromFile(rootBuildGradlePth)
	if err != nil {
		return nil, err
	}

	return androidGradlePluginVersionFromContent(content)
}

// EnsureAndroidLicences ...
func EnsureAndroidLicences(androidHome string) error {
	licenceMap := map[string]string{
		"android-sdk-license":         "\n8933bad161af4178b1185d1a37fbf41ea5269c55",
		"android-sdk-preview-license": "\n84831b9409646a918e30573bab4c9c91346d8abd",
		"intel-android-extra-license": "\nd975f751698a77b662f1254ddbeed3901e976f5a",
	}

	licencesDir := filepath.Join(androidHome, "licenses")
	if exist, err := pathutil.IsDirExists(licencesDir); err != nil {
		return err
	} else if !exist {
		if err := os.Mkdir(licencesDir, os.ModePerm); err != nil {
			return err
		}
	}

	for name, content := range licenceMap {
		pth := filepath.Join(licencesDir, name)

		if exist, err := pathutil.IsPathExists(pth); err != nil {
			return err
		} else if !exist {
			if err := fileutil.WriteStringToFile(pth, content); err != nil {
				return err
			}
		}
	}

	return nil
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

// -----------------------
// --- Main
// -----------------------

func main() {
	//
	// Input validation
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		failf("Issue with input: %s", err)
	}

	//
	fmt.Println()
	log.Infof("Step determined configs:")

	rootBuildGradleFile, err := pathutil.AbsPath(configs.RootBuildGradleFile)
	if err != nil {
		failf("Failed to expand root build.gradle file path (%s), error: %s", configs.RootBuildGradleFile, err)
	}

	log.Printf("- root build.gradle file: %s", rootBuildGradleFile)

	pluginVersion, err := AndroidGradlePluginVersion(rootBuildGradleFile)
	if err != nil {
		failf("Failed to determine android gradle plugin version, error: %s", err)
	}

	log.Printf("- android gradle plugin version: %s", pluginVersion)

	if !pluginVersion.LessThan(AutoDownloadSDKComponentsMinGradlePluginVersion) {
		fmt.Println()
		log.Infof("Android Gradle Plugin version: %s will auto-download missing sdk components with Gradle", pluginVersion.String())

		if err := EnsureAndroidLicences(configs.AndroidHome); err != nil {
			failf("Failed to ensure android licences, error: %s", err)
		}

		cmd := command.New("./gradlew", "dependencies")
		cmd.SetStdin(strings.NewReader("y"))
		cmd.SetDir(filepath.Dir(configs.GradlewPath))

		log.Printf("Ensure sdk components using: $ %s", cmd.PrintableCommandArgs())

		if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
			log.Errorf("Command failed with output:")
			log.Printf(out)
			failf("%s", err)
		} else {
			log.Printf(out)
		}
	} else {
		fmt.Println()
		log.Warnf("Android Gradle Plugin version: %s will NOT auto-download missing sdk components", pluginVersion.String())

		if err := EnsureAndroidLicences(configs.AndroidHome); err != nil {
			failf("Failed to ensure android licences, error: %s", err)
		}

		androidSdk, err := sdk.New(configs.AndroidHome)
		if err != nil {
			failf("Failed to create sdk, error: %s", err)
		}

		sdkManager, err := sdkmanager.New(androidSdk)
		if err != nil {
			failf("Failed to create sdk manager, error: %s", err)
		}

		retry := true
		for retry {
			gradleCmd := command.New("./gradlew", "dependencies")
			gradleCmd.SetStdin(strings.NewReader("y"))
			gradleCmd.SetDir(filepath.Dir(configs.GradlewPath))

			fmt.Println()
			log.Printf("Searching for missing sdk components using:")
			log.Printf("$ %s", gradleCmd.PrintableCommandArgs())

			if out, err := gradleCmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
				reader := strings.NewReader(out)
				scanner := bufio.NewScanner(reader)

				missingSDKComponentFound := false

				for scanner.Scan() {
					line := scanner.Text()

					{
						// failed to find target with hash string 'android-22'
						targetPattern := `failed to find target with hash string 'android-(?P<version>.*)'\s*`
						targetRe := regexp.MustCompile(targetPattern)
						if matches := targetRe.FindStringSubmatch(line); len(matches) == 2 {
							missingSDKComponentFound = true

							targetVersion := "android-" + matches[1]

							log.Warnf("Missing platform version found: %s", targetVersion)

							platformComponent := sdkcomponent.Platform{
								Version: targetVersion,
							}

							cmd := sdkManager.InstallCommand(platformComponent)
							cmd.SetStdin(strings.NewReader("y"))

							log.Printf("Installing platform version using:")
							log.Printf("$ %s", cmd.PrintableCommandArgs())

							if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
								log.Errorf("Command failed with output:")
								log.Printf(out)
								failf("%s", err)
							}
						}
					}

					{
						// failed to find Build Tools revision 22.0.1
						buildToolsPattern := `failed to find Build Tools revision (?P<version>[0-9.]*)\s*`
						buildToolsRe := regexp.MustCompile(buildToolsPattern)
						if matches := buildToolsRe.FindStringSubmatch(line); len(matches) == 2 {
							missingSDKComponentFound = true

							buildToolsVersion := matches[1]

							log.Warnf("Missing build tools version found: %s", buildToolsVersion)

							buildToolsComponent := sdkcomponent.BuildTool{
								Version: buildToolsVersion,
							}

							cmd := sdkManager.InstallCommand(buildToolsComponent)
							cmd.SetStdin(strings.NewReader("y"))

							log.Printf("Installing build tools version using:")
							log.Printf("$ %s", cmd.PrintableCommandArgs())

							if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
								log.Errorf("Command failed with output:")
								log.Printf(out)
								failf("%s", err)
							}
						}
					}
				}

				if err := scanner.Err(); err != nil {
					failf("failed to analyze gradle output, error: %s", err)
				}

				if !missingSDKComponentFound {
					log.Printf(out)
					failf("%s", err)
				}
			} else {
				retry = false
			}
		}
	}

	fmt.Println()
	log.Donef("Required SDK components are installed")
}
