package sdk

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-android/sdkcomponent"
	"github.com/bitrise-tools/go-android/sdkmanager"
)

// Model ...
type Model struct {
	buildTools   []sdkcomponent.BuildTool
	platforms    []sdkcomponent.Platform
	systemImages []sdkcomponent.SystemImage
}

// New ...
func New(androidHome string) (*Model, error) {
	binPth := filepath.Join(androidHome, "tools", "bin", "sdkmanager")

	legacy, err := sdkmanager.IsLegacySDKManager(androidHome)
	if err != nil {
		return nil, err
	} else if legacy {
		binPth = filepath.Join(androidHome, "tools", "android")
	}

	if exist, err := pathutil.IsPathExists(binPth); err != nil {
		return nil, err
	} else if !exist {
		return nil, fmt.Errorf("no sdk manager tool found at: %s", binPth)
	}

	var cmd *command.Model
	if legacy {
		cmd = command.New(binPth, "list", "sdk", "--all", "--extended")
	} else {
		cmd = command.New(binPth, "--list", "--verbose")
	}

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return nil, err
	}

	if legacy {
		return parseLegacySDKOut(out)
	}
	return parseSDKOut(out)
}

func (sdk *Model) addBuildTool(buildTool sdkcomponent.BuildTool) {
	if sdk.buildTools == nil {
		sdk.buildTools = []sdkcomponent.BuildTool{}
	}
	sdk.buildTools = append(sdk.buildTools, buildTool)
}

func (sdk *Model) addPlatform(platform sdkcomponent.Platform) {
	if sdk.platforms == nil {
		sdk.platforms = []sdkcomponent.Platform{}
	}
	sdk.platforms = append(sdk.platforms, platform)
}

func (sdk *Model) addSystemImage(systemImage sdkcomponent.SystemImage) {
	if sdk.systemImages == nil {
		sdk.systemImages = []sdkcomponent.SystemImage{}
	}
	sdk.systemImages = append(sdk.systemImages, systemImage)
}

// GetBuildTool ...
func (sdk *Model) GetBuildTool(version string) (sdkcomponent.BuildTool, bool) {
	for _, buildTool := range sdk.buildTools {
		if buildTool.Version == version {
			return buildTool, true
		}
	}
	return sdkcomponent.BuildTool{}, false
}

// GetPlatform ...
func (sdk *Model) GetPlatform(version string) (sdkcomponent.Platform, bool) {
	for _, platform := range sdk.platforms {
		if platform.Version == version {
			return platform, true
		}
	}
	return sdkcomponent.Platform{}, false
}

// GetSystemImage ...
func (sdk *Model) GetSystemImage(platform string, abi string, tag string) (sdkcomponent.SystemImage, bool) {
	systemImages := []sdkcomponent.SystemImage{}

	for _, systemImage := range sdk.systemImages {
		if systemImage.Platform == platform && systemImage.ABI == abi {
			systemImages = append(systemImages, systemImage)
		}
	}

	defaultTag := tag
	if tag == "" {
		defaultTag = "default"
	}

	for _, systemImage := range systemImages {
		if systemImage.Tag == defaultTag {
			return systemImage, true
		}
	}

	defaultTag = "google_apis"

	for _, systemImage := range systemImages {
		if systemImage.Tag == defaultTag {
			return systemImage, true
		}
	}

	return sdkcomponent.SystemImage{}, false
}

func parseSDKOut(sdkOut string) (*Model, error) {
	/*
		build-tools;19.1.0
			Description:        Android SDK Build-Tools 19.1
			Version:            19.1.0

		platforms;android-10
			Description:        Android SDK Platform 10
			Version:            2

		system-images;android-16;google_apis;armeabi-v7a
			Description:        Google APIs ARM EABI v7a System Image
			Version:            5
	*/

	sectionSeparator := "--------------------------------------"

	installedPackagesStartPattern := "Installed packages:"
	installedPackagesSection := false

	availablePackagesStartPattern := "Available Packages:"
	availablePackagesSection := false

	availableUpdatesStartPattern := "Available Updates:"

	sdk := Model{}

	reader := strings.NewReader(sdkOut)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if line == sectionSeparator {
			continue
		}

		if strings.HasPrefix(line, " ") {
			continue
		}

		//

		if trimmedLine == installedPackagesStartPattern {
			installedPackagesSection = true
			availablePackagesSection = false
			continue
		}

		if trimmedLine == availablePackagesStartPattern {
			installedPackagesSection = false
			availablePackagesSection = true
			continue
		}

		if trimmedLine == availableUpdatesStartPattern {
			installedPackagesSection = false
			availablePackagesSection = false
			continue
		}

		if !installedPackagesSection && !availablePackagesSection {
			continue
		}

		//

		if strings.HasPrefix(trimmedLine, "build-tools") {
			split := strings.Split(trimmedLine, ";")
			if len(split) == 2 {
				sdk.addBuildTool(sdkcomponent.BuildTool{
					Version:      split[1],
					SDKStylePath: trimmedLine,
				})
			}
		}

		if strings.HasPrefix(trimmedLine, "platforms") {
			split := strings.Split(trimmedLine, ";")
			if len(split) == 2 {
				sdk.addPlatform(sdkcomponent.Platform{
					Version:      split[1],
					SDKStylePath: trimmedLine,
				})
			}
		}

		if strings.HasPrefix(trimmedLine, "system-images") {
			split := strings.Split(trimmedLine, ";")
			if len(split) == 4 {
				sdk.addSystemImage(sdkcomponent.SystemImage{
					Platform:     split[1],
					Tag:          split[2],
					ABI:          split[3],
					SDKStylePath: trimmedLine,
				})
			}
		}

	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &sdk, nil
}

func parseLegacySDKOut(sdkOut string) (*Model, error) {
	/*
		----------
		id: 23 or "build-tools-19.1.0"
			Type: BuildTool
			Desc: Android SDK Build-tools, revision 19.1
		----------
		id: 48 or "android-10"
			Type: Platform
			Desc: Android SDK Platform 10
				Revision 2
		----------
		id: 114 or "sys-img-armeabi-v7a-google_apis-16"
			Type: SystemImage
			Desc: Google APIs ARM EABI v7a System Image
				Revision 5
				Requires SDK Platform Android API 16
		----------
		id: 123 or "sys-img-x86-android-10"
			Type: SystemImage
			Desc: Intel x86 Atom System Image
				Revision 4
				Requires SDK Platform Android API 10
		----------
	*/

	sectionSeparator := `----------`
	legacySDKStyleNamePatter := `"(?P<name>.*)"`
	androidTVPattern := `sys-img-(?P<abi>.*)-android-tv-(?P<platform>.*)`
	androidWearPattern := `sys-img-(?P<abi>.*)-android-wear-(?P<platform>.*)`
	googleAPIpattern := `sys-img-(?P<abi>.*)-google_apis-(?P<platform>.*)`
	androidPattern := `sys-img-(?P<abi>.*)-android-(?P<platform>.*)`

	sdk := Model{}

	reader := strings.NewReader(sdkOut)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == sectionSeparator {
			continue
		}

		legacySDKSytleName := ""
		if match := regexp.MustCompile(legacySDKStyleNamePatter).FindStringSubmatch(trimmedLine); len(match) == 2 {
			legacySDKSytleName = match[1]
		}

		if legacySDKSytleName == "" {
			continue
		}

		if strings.HasPrefix(legacySDKSytleName, "build-tools-") {
			sdk.addBuildTool(sdkcomponent.BuildTool{
				Version:            strings.TrimPrefix(legacySDKSytleName, "build-tools-"),
				LegacySDKStylePath: legacySDKSytleName,
			})
		}

		if strings.HasPrefix(legacySDKSytleName, "android-") {
			sdk.addPlatform(sdkcomponent.Platform{
				Version:            legacySDKSytleName,
				LegacySDKStylePath: legacySDKSytleName,
			})
		}

		if strings.HasPrefix(legacySDKSytleName, "sys-img-") {
			if strings.Contains(legacySDKSytleName, "-android-tv-") {
				if match := regexp.MustCompile(androidTVPattern).FindStringSubmatch(legacySDKSytleName); len(match) == 3 {
					sdk.addSystemImage(sdkcomponent.SystemImage{
						Platform:           "android-" + match[2],
						Tag:                "android-tv",
						ABI:                match[1],
						LegacySDKStylePath: legacySDKSytleName,
					})
					continue
				}
			}

			if strings.Contains(legacySDKSytleName, "-android-wear-") {
				if match := regexp.MustCompile(androidWearPattern).FindStringSubmatch(legacySDKSytleName); len(match) == 3 {
					sdk.addSystemImage(sdkcomponent.SystemImage{
						Platform:           "android-" + match[2],
						Tag:                "android-wear",
						ABI:                match[1],
						LegacySDKStylePath: legacySDKSytleName,
					})
					continue
				}
			}

			if strings.Contains(legacySDKSytleName, "-google_apis-") {
				if match := regexp.MustCompile(googleAPIpattern).FindStringSubmatch(legacySDKSytleName); len(match) == 3 {
					sdk.addSystemImage(sdkcomponent.SystemImage{
						Platform:           "android-" + match[2],
						Tag:                "google_apis",
						ABI:                match[1],
						LegacySDKStylePath: legacySDKSytleName,
					})
					continue
				}
			}

			if strings.Contains(legacySDKSytleName, "-android-") {
				if match := regexp.MustCompile(androidPattern).FindStringSubmatch(legacySDKSytleName); len(match) == 3 {
					sdk.addSystemImage(sdkcomponent.SystemImage{
						Platform:           "android-" + match[2],
						Tag:                "android",
						ABI:                match[1],
						LegacySDKStylePath: legacySDKSytleName,
					})
					continue
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &sdk, nil
}
