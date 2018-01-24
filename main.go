package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-tools/go-android/sdk"
	"github.com/bitrise-tools/go-android/sdkcomponent"
	"github.com/bitrise-tools/go-android/sdkmanager"
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

func ensureAndroidLicences(androidHome string, isLegacySDK bool) error {
	if !isLegacySDK {
		licensesCmd := command.New(filepath.Join(androidHome, "tools/bin/sdkmanager"), "--licenses")
		licensesCmd.SetStdin(bytes.NewReader([]byte(strings.Repeat("y\n", 1000))))
		if err := licensesCmd.Run(); err != nil {
			log.Warnf("Failed to install licenses using $(sdkmanager --licenses) command")
			log.Printf("Continue using legacy license installation...")
			fmt.Println()
		} else {
			return nil
		}
	}

	licenceMap := map[string]string{
		"android-sdk-license":           "8933bad161af4178b1185d1a37fbf41ea5269c55\n\nd56f5187479451eabf01fb78af6dfcb131a6481e",
		"android-googletv-license":      "\n601085b94cd77f0b54ff86406957099ebe79c4d6",
		"android-sdk-preview-license":   "\n84831b9409646a918e30573bab4c9c91346d8abd",
		"intel-android-extra-license":   "\nd975f751698a77b662f1254ddbeed3901e976f5a",
		"google-gdk-license":            "\n33b6a2b64607f11b759f320ef9dff4ae5c47d97a",
		"mips-android-sysimage-license": "\ne9acab5b5fbb560a72cfaecce8946896ff6aab9d",
	}

	licencesDir := filepath.Join(androidHome, "licenses")
	if exist, err := pathutil.IsDirExists(licencesDir); err != nil {
		return err
	} else if !exist {
		if err := os.MkdirAll(licencesDir, os.ModePerm); err != nil {
			return err
		}
	}

	for name, content := range licenceMap {
		pth := filepath.Join(licencesDir, name)

		if err := fileutil.WriteStringToFile(pth, content); err != nil {
			return err
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
	// Input validation
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		failf("Issue with input: %s", err)
	}

	fmt.Println()
	log.Infof("Preparation")

	// Set executable permission for gradlew
	log.Printf("Set executable permission for gradlew")
	if err := os.Chmod(configs.GradlewPath, 0770); err != nil {
		failf("Failed to set executable permission for gradlew, error: %s", err)
	}

	// Initialize Android SDK
	log.Printf("Initialize Android SDK")
	androidSdk, err := sdk.New(configs.AndroidHome)
	if err != nil {
		failf("Failed to initialize Android SDK, error: %s", err)
	}

	sdkManager, err := sdkmanager.New(androidSdk)
	if err != nil {
		failf("Failed to create SDK manager, error: %s", err)
	}

	// Ensure android licences
	log.Printf("Ensure android licences")
	if err := ensureAndroidLicences(configs.AndroidHome, sdkManager.IsLegacySDK()); err != nil {
		failf("Failed to ensure android licences, error: %s", err)
	}

	// Ensure required Android SDK components
	fmt.Println()
	log.Infof("Ensure required Android SDK components")

	retryCount := 0
	for true {
		gradleCmd := command.New("./gradlew", "dependencies")
		gradleCmd.SetStdin(strings.NewReader("y"))
		gradleCmd.SetDir(filepath.Dir(configs.GradlewPath))

		log.Printf("Searching for missing SDK components using:")
		log.Printf("$ %s", gradleCmd.PrintableCommandArgs())

		if out, err := gradleCmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
			reader := strings.NewReader(out)
			scanner := bufio.NewScanner(reader)

			missingSDKComponentFound := false

			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println(line)
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

						if err := retry.Times(1).Wait(time.Second).Try(func(attempt uint) error {
							if attempt > 0 {
								log.Warnf("Retrying...")
							}

							if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
								if attempt > 0 {
									return fmt.Errorf("output: %s, error: %s", out, err)
								}
								return err
							}

							return nil
						}); err != nil {
							log.Errorf("Failed to install platform:")
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

						if err := retry.Times(1).Wait(time.Second).Try(func(attempt uint) error {
							if attempt > 0 {
								log.Warnf("Retrying...")
							}

							if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
								if attempt > 0 {
									return fmt.Errorf("output: %s, error: %s", out, err)
								}
								return err
							}

							return nil
						}); err != nil {
							log.Errorf("Failed to install build tools:")
							failf("%s", err)
						}
					}
				}

				{
					// Example: "Could not find com.android.support.constraint:constraint-layout:1.0.2."
					extrasPattern := `Could not find (?P<package>com\.android\.support\..*)\.`
					extrasRe := regexp.MustCompile(extrasPattern)
					if matches := extrasRe.FindStringSubmatch(line); len(matches) == 2 {
						missingSDKComponentFound = true

						log.Warnf("Missing extras library found: %s", matches[1])

						lib := matches[1]
						firstColon := strings.Index(lib, ":")
						lib = strings.Replace(lib[:firstColon], ".", ";", -1) + strings.Replace(lib[firstColon:], ":", ";", -1)

						extrasComponents := sdkcomponent.SupportLibraryInstallComponents()
						extrasComponents = append(extrasComponents, sdkcomponent.Extras{
							Provider:    "m2repository",
							PackageName: lib,
						})
						for _, e := range extrasComponents {
							cmd := sdkManager.InstallCommand(e)
							cmd.SetStdin(strings.NewReader("y"))

							log.Printf("Installing extras using:")
							log.Printf("$ %s", cmd.PrintableCommandArgs())

							if err := retry.Times(1).Wait(time.Second).Try(func(attempt uint) error {
								if attempt > 0 {
									log.Warnf("Retrying...")
								}

								if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
									if attempt > 0 {
										return fmt.Errorf("output: %s, error: %s", out, err)
									}
									return err
								}

								return nil
							}); err != nil {
								log.Errorf("Failed to install build tools:")
								failf("%s", err)
							}
						}
					}
				}
			}

			if err := scanner.Err(); err != nil {
				failf("failed to analyze gradle output, error: %s", err)
			}

			if !missingSDKComponentFound {
				if retryCount < 2 {
					log.Errorf("Failed to find missing components, retrying...")
					retryCount++
					continue
				}
				log.Printf(out)
				failf("%s", err)
			}
		} else {
			break
		}
	}

	fmt.Println()
	log.Donef("Required SDK components are installed")
}
