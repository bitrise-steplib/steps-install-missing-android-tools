package androidcomponents

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-android/sdk"
	"github.com/bitrise-io/go-android/sdkcomponent"
	"github.com/bitrise-io/go-android/sdkmanager"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/env"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/sliceutil"
)

var logger = log.NewLogger()
var cmdFactory = command.NewFactory(env.NewRepository())

type installer struct {
	androidSDK  *sdk.Model
	sdkManager  *sdkmanager.Model
	gradlewPath string
}

// InstallLicences ...
func InstallLicences(androidSdk *sdk.Model) error {
	sdkManager, err := sdkmanager.New(androidSdk, cmdFactory)
	if err != nil {
		return err
	}

	licencesDir, licenceMap := filepath.Join(androidSdk.GetAndroidHome(), "licenses"), map[string]string{
		"android-sdk-license":           "\n24333f8a63b6825ea9c5514f83c2829b004d1fee",
		"android-googletv-license":      "\n601085b94cd77f0b54ff86406957099ebe79c4d6",
		"android-sdk-preview-license":   "\n84831b9409646a918e30573bab4c9c91346d8abd",
		"intel-android-extra-license":   "\nd975f751698a77b662f1254ddbeed3901e976f5a",
		"google-gdk-license":            "\n33b6a2b64607f11b759f320ef9dff4ae5c47d97a",
		"mips-android-sysimage-license": "\ne9acab5b5fbb560a72cfaecce8946896ff6aab9d",
	}

	if !sdkManager.IsLegacySDK() {
		cmdOpts := command.Opts{
			Stdin: bytes.NewReader([]byte(strings.Repeat("y\n", 1000))),
		}
		licensesCmd := cmdFactory.Create(filepath.Join(androidSdk.GetAndroidHome(), "tools/bin/sdkmanager"), []string{"--licenses"}, &cmdOpts)
		if err := licensesCmd.Run(); err != nil {
			logger.Warnf("Failed to install licenses using $(sdkmanager --licenses) command")
			logger.Printf("Continue using legacy license installation...")
			logger.Printf("")
		} else {
			sdkLicencePath, oldLicenceHash := filepath.Join(licencesDir, "android-sdk-license"), "d56f5187479451eabf01fb78af6dfcb131a6481e"
			if content, err := fileutil.ReadStringFromFile(sdkLicencePath); err == nil && strings.Contains(content, oldLicenceHash) {
				if err := fileutil.WriteStringToFile(sdkLicencePath, licenceMap[filepath.Base(sdkLicencePath)]); err != nil {
					return err
				}
			}
			return nil
		}
	}

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

// Ensure ...
func Ensure(androidSdk *sdk.Model, gradlewPath string) error {
	sdkManager, err := sdkmanager.New(androidSdk, cmdFactory)
	if err != nil {
		return err
	}
	i := installer{
		androidSdk,
		sdkManager,
		gradlewPath,
	}

	return retry.Times(1).Wait(time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			logger.Warnf("Retrying...")
		}
		return i.scanDependencies()
	})
}

func (i installer) getDependencyCases() map[string]func(match string) error {
	return map[string]func(match string) error{
		`failed to find target with hash string 'android-(.*)'\s*`:            i.target,
		`failed to find Build Tools revision ([0-9.]*)\s*`:                    i.buildTool,
		`Could not find (com\.android\.support:.*)\.`:                         i.extrasLib,
		`Could not find any version that matches (com\.android\.support:*)\.`: i.extrasLib,
	}
}

func getDependenciesOutput(projectLocation string) (string, error) {
	cmdOpts := command.Opts{
		Stdin: strings.NewReader("y"),
		Dir:   projectLocation,
	}
	gradleCmd := cmdFactory.Create("./gradlew", []string{"dependencies", "--stacktrace"}, &cmdOpts)
	return gradleCmd.RunAndReturnTrimmedCombinedOutput()
}

func (i installer) scanDependencies(foundMatches ...string) error {
	out, err := getDependenciesOutput(filepath.Dir(i.gradlewPath))
	if err == nil {
		return nil
	}
	err = fmt.Errorf("output: %s\nerror: %s", out, err)
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		for pattern, callback := range i.getDependencyCases() {
			re := regexp.MustCompile(pattern)
			if matches := re.FindStringSubmatch(line); len(matches) == 2 {
				if sliceutil.IsStringInSlice(matches[1], foundMatches) {
					return fmt.Errorf("unable to solve a dependency installation for the output:\n%s", out)
				}
				if callbackErr := callback(matches[1]); callbackErr != nil {
					logger.Printf(out)
					return callbackErr
				}
				err = nil
				return i.scanDependencies(append(foundMatches, matches[1])...)
			}
		}
	}
	if scanner.Err() != nil {
		logger.Printf(out)
		return scanner.Err()
	}
	return err
}

func (i installer) target(version string) error {
	logger.Warnf("Missing platform version found: %s", version)

	version = "android-" + version
	platformComponent := sdkcomponent.Platform{
		Version: version,
	}
	cmd := i.sdkManager.InstallCommand(platformComponent)

	logger.Printf("Installing platform version using:")
	logger.Printf("$ %s", cmd.PrintableCommandArgs())

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("output: %s, error: %s", out, err)
	}
	return nil
}

func (i installer) buildTool(buildToolsVersion string) error {
	logger.Warnf("Missing build tools version found: %s", buildToolsVersion)

	buildToolsComponent := sdkcomponent.BuildTool{
		Version: buildToolsVersion,
	}

	cmd := i.sdkManager.InstallCommand(buildToolsComponent)

	logger.Printf("Installing build tools version using:")
	logger.Printf("$ %s", cmd.PrintableCommandArgs())

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("output: %s, error: %s", out, err)
	}
	return nil
}

func (i installer) extrasLib(lib string) error {
	logger.Warnf("Missing extras library found: %s", lib)

	firstColon := strings.Index(lib, ":")
	lib = strings.Replace(lib[:firstColon], ".", ";", -1) + strings.Replace(lib[firstColon:], ":", ";", -1)

	extrasComponents := sdkcomponent.SupportLibraryInstallComponents()
	extrasComponents = append(extrasComponents, sdkcomponent.Extras{
		Provider:    "m2repository",
		PackageName: lib,
	})
	for _, e := range extrasComponents {
		cmd := i.sdkManager.InstallCommand(e)

		logger.Printf("Installing extras using:")
		logger.Printf("$ %s", cmd.PrintableCommandArgs())

		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			return fmt.Errorf("output: %s, error: %s", out, err)
		}
		return nil
	}
	return nil
}
