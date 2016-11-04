package installer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/depman/pathutil"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/hashicorp/go-version"
)

// -----------------------
// --- Stucts
// -----------------------

// AndroidToolHelperModel ...
type AndroidToolHelperModel struct {
	androidHome string
}

// NewAndroidToolHelperModel ...
func NewAndroidToolHelperModel() (AndroidToolHelperModel, error) {
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome == "" {
		return AndroidToolHelperModel{}, fmt.Errorf("Missing ANDROID_HOME environment")
	}

	return AndroidToolHelperModel{
		androidHome: androidHome,
	}, nil
}

// IsSDKVersionInstalled ...
func (androidToolHelper AndroidToolHelperModel) IsSDKVersionInstalled(v *version.Version) (bool, error) {
	// $ANDROID_HOME/platforms/android-23
	sdkMajorVersion := v.Segments()[0]

	sdkFolderName := fmt.Sprintf("android-%d", sdkMajorVersion)
	sdkPath := filepath.Join(androidToolHelper.androidHome, "platforms", sdkFolderName)

	exist, err := pathutil.IsPathExists(sdkPath)
	if err != nil {
		return false, err
	}

	return exist, nil
}

// IsBuildToolsInstalled ...
func (androidToolHelper AndroidToolHelperModel) IsBuildToolsInstalled(v *version.Version) (bool, error) {
	// $ANDROID_HOME/build-tools/23.0.3
	buildToolsPath := filepath.Join(androidToolHelper.androidHome, "build-tools", v.String())

	exist, err := pathutil.IsPathExists(buildToolsPath)
	if err != nil {
		return false, err
	}

	return exist, nil
}

// IsSupportLibraryInstalled ...
func (androidToolHelper AndroidToolHelperModel) IsSupportLibraryInstalled() (bool, error) {
	// $ANDROID_HOME/extras/android/support
	supportLibraryPath := filepath.Join(androidToolHelper.androidHome, "extras", "android", "support")

	exist, err := pathutil.IsPathExists(supportLibraryPath)
	if err != nil {
		return false, err
	}

	return exist, nil
}

// InstallSDKVersion ...
func (androidToolHelper AndroidToolHelperModel) InstallSDKVersion(v *version.Version) error {
	// $ANDROID_HOME/platforms/android-23
	sdkMajorVersion := v.Segments()[0]
	sdkMajorVersionStr := string(sdkMajorVersion)
	sdkFilter := "android-" + sdkMajorVersionStr
	cmdSlice := androidInstallCmdSlice(sdkFilter)
	return runAndroidInstallCmdSlice(cmdSlice)
}

// InstallBuildToolsVersion ...
func (androidToolHelper AndroidToolHelperModel) InstallBuildToolsVersion(v *version.Version) error {
	// $ANDROID_HOME/build-tools/23.0.3
	sdkFilter := "build-tools-" + v.String()
	cmdSlice := androidInstallCmdSlice(sdkFilter)
	return runAndroidInstallCmdSlice(cmdSlice)
}

// UpdateSupportLibrary ...
func (androidToolHelper AndroidToolHelperModel) UpdateSupportLibrary() error {
	// $ANDROID_HOME/extras/android/support
	platformToolsFilter := "platform-tools"
	platformToolsCmdSlice := androidInstallCmdSlice(platformToolsFilter)
	if err := runAndroidInstallCmdSlice(platformToolsCmdSlice); err != nil {
		return err
	}

	supportLibraryFilter := "extra-android-support"
	supportLibraryCmdSlice := androidInstallCmdSlice(supportLibraryFilter)
	return runAndroidInstallCmdSlice(supportLibraryCmdSlice)
}

// UpdateGooglePlayServices ...
func (androidToolHelper AndroidToolHelperModel) UpdateGooglePlayServices() error {
	// $ANDROID_HOME/extras/google/google_play_services
	androidM2RepositoryFilter := "extra-android-m2repository"
	androidM2RepositoryCmdSlice := androidInstallCmdSlice(androidM2RepositoryFilter)
	if err := runAndroidInstallCmdSlice(androidM2RepositoryCmdSlice); err != nil {
		return err
	}

	googleM2RepositoryFilter := "extra-google-m2repository"
	googleM2RepositoryCmdSlice := androidInstallCmdSlice(googleM2RepositoryFilter)
	if err := runAndroidInstallCmdSlice(googleM2RepositoryCmdSlice); err != nil {
		return err
	}

	googlePlayServicesFilter := "extra-google-google_play_services"
	googlePlayServicesCmdSlice := androidInstallCmdSlice(googlePlayServicesFilter)
	return runAndroidInstallCmdSlice(googlePlayServicesCmdSlice)
}

// IsGooglePlayServicesInstalled ...
func (androidToolHelper AndroidToolHelperModel) IsGooglePlayServicesInstalled() (bool, error) {
	// $ANDROID_HOME/extras/google/google_play_services
	googlePlayServicesPath := filepath.Join(androidToolHelper.androidHome, "extras", "google", "google_play_services")

	exist, err := pathutil.IsPathExists(googlePlayServicesPath)
	if err != nil {
		return false, err
	}

	return exist, nil
}

// -----------------------
// --- Functions
// -----------------------

func androidInstallCmdSlice(filter string) []string {
	return []string{
		"android",
		"update",
		"sdk",
		"--no-ui",
		"--all",
		"--filter",
		filter,
	}
}

func isInstallSuccess(output string) (bool, error) {
	installSuccessRegexp := regexp.MustCompile(`\s*Done. [0-9]+ package installed.`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		matche := installSuccessRegexp.FindString(scanner.Text())
		if matche != "" {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func runAndroidInstallCmdSlice(cmdSlice []string) error {
	cmdStr := cmdex.LogPrintableCommandArgs(cmdSlice)
	log.Detail(cmdStr)

	var outBuffer bytes.Buffer
	outWriter := io.Writer(&outBuffer)

	var errBuffer bytes.Buffer
	errWriter := io.Writer(&errBuffer)

	inputReader := strings.NewReader("y")

	err := cmdex.RunCommandWithReaderAndWriters(inputReader, outWriter, errWriter, cmdSlice[0], cmdSlice[1:]...)
	errorStr := string(errBuffer.Bytes())
	outStr := string(outBuffer.Bytes())

	if err != nil {
		if !errorutil.IsExitStatusError(err) {
			return err
		}

		if errorStr != "" && errorutil.IsExitStatusErrorStr(errorStr) {
			return errors.New(errorStr)
		}

		return errors.New(outStr)
	}

	success, err := isInstallSuccess(outStr)
	if err != nil {
		return fmt.Errorf("failed to check if install success, error: %s", err)
	}

	if !success {
		return fmt.Errorf("install failed, output: %s", outStr)
	}

	return nil
}
