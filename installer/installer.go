package installer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"strconv"

	"github.com/bitrise-io/depman/pathutil"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/hashicorp/go-version"
)

// -----------------------
// --- Stucts
// -----------------------

// Model ...
type Model struct {
	androidHome string
}

// New ...
func New(androidHome string) Model {
	return Model{
		androidHome: androidHome,
	}
}

// IsSDKVersionInstalled ...
func (m Model) IsSDKVersionInstalled(v *version.Version) (bool, error) {
	// $ANDROID_HOME/platforms/android-23
	sdkMajorVersion := v.Segments()[0]

	sdkFolderName := fmt.Sprintf("android-%d", sdkMajorVersion)
	sdkPath := filepath.Join(m.androidHome, "platforms", sdkFolderName)

	return pathutil.IsPathExists(sdkPath)
}

// IsBuildToolsInstalled ...
func (m Model) IsBuildToolsInstalled(v *version.Version) (bool, error) {
	// $ANDROID_HOME/build-tools/23.0.3
	buildToolsPath := filepath.Join(m.androidHome, "build-tools", v.String())

	return pathutil.IsPathExists(buildToolsPath)
}

// IsSupportLibraryInstalled ...
func (m Model) IsSupportLibraryInstalled() (bool, error) {
	// $ANDROID_HOME/extras/android/m2repository/com/android/support
	supportLibraryPath := filepath.Join(m.androidHome, "extras/android/m2repository/com/android/support")

	return pathutil.IsPathExists(supportLibraryPath)
}

// IsGooglePlayServicesInstalled ...
func (m Model) IsGooglePlayServicesInstalled() (bool, error) {
	// $ANDROID_HOME/extras/google/google_play_services
	googlePlayServicesPath := filepath.Join(m.androidHome, "extras/google/google_play_services")

	return pathutil.IsPathExists(googlePlayServicesPath)
}

// InstallSDKVersion ...
func (m Model) InstallSDKVersion(v *version.Version) error {
	/*
		id: 33 or "android-25"
			Type: Platform
			Desc: Android SDK Platform 25
				Revision 3
	*/

	// $ANDROID_HOME/platforms/android-23
	sdkMajorVersion := v.Segments()[0]
	sdkMajorVersionStr := strconv.Itoa(sdkMajorVersion)

	sdkFilter := "android-" + sdkMajorVersionStr
	cmdSlice := androidInstallCmdSlice(sdkFilter)

	return runAndroidInstallCmdSlice(cmdSlice)
}

// InstallBuildToolsVersion ...
func (m Model) InstallBuildToolsVersion(v *version.Version) error {
	/*
		id: 3 or "build-tools-25.0.2"
			Type: BuildTool
			Desc: Android SDK Build-tools, revision 25.0.2
	*/

	// $ANDROID_HOME/build-tools/23.0.3
	sdkFilter := "build-tools-" + v.String()
	cmdSlice := androidInstallCmdSlice(sdkFilter)

	return runAndroidInstallCmdSlice(cmdSlice)
}

// UpdateSupportLibrary ...
func (m Model) UpdateSupportLibrary() error {
	/*
		id: 166 or "extra-android-m2repository"
			Type: Extra
			Desc: Android Support Repository, revision 41
				By Android
				Local Maven repository for Support Libraries
				Install path: extras/android/m2reposito
	*/

	// $ANDROID_HOME/extras/android/m2repository/com/android/support
	androidM2RepositoryFilter := "extra-android-m2repository"
	androidM2RepositoryCmdSlice := androidInstallCmdSlice(androidM2RepositoryFilter)

	if err := runAndroidInstallCmdSlice(androidM2RepositoryCmdSlice); err != nil {
		return err
	}

	/*
		id: 173 or "extra-google-m2repository"
			Type: Extra
			Desc: Google Repository, revision 41
				By Google Inc.
				Local Maven repository for Support Libraries
				Install path: extras/google/m2repository
	*/

	// $ANDROID_HOME/extras/google/m2repository/com/google/android/support
	googleM2RepositoryFilter := "extra-google-m2repository"
	googleM2RepositoryCmdSlice := androidInstallCmdSlice(googleM2RepositoryFilter)

	return runAndroidInstallCmdSlice(googleM2RepositoryCmdSlice)
}

// UpdateGooglePlayServices ...
func (m Model) UpdateGooglePlayServices() error {
	/*
		id: 172 or "extra-google-google_play_services"
			Type: Extra
			Desc: Google Play services, revision 38
				By Google Inc.
				Google Play services Javadocs and sample code
				Install path: extras/google/google_play_services
	*/

	// $ANDROID_HOME/extras/google/google_play_services
	googlePlayServicesFilter := "extra-google-google_play_services"
	googlePlayServicesCmdSlice := androidInstallCmdSlice(googlePlayServicesFilter)

	return runAndroidInstallCmdSlice(googlePlayServicesCmdSlice)
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
	log.Printf("$ %s", command.PrintableCommandArgs(false, cmdSlice))

	var outBuffer bytes.Buffer
	outWriter := io.Writer(&outBuffer)

	var errBuffer bytes.Buffer
	errWriter := io.Writer(&errBuffer)

	inputReader := strings.NewReader("y")

	err := command.RunCommandWithReaderAndWriters(inputReader, outWriter, errWriter, cmdSlice[0], cmdSlice[1:]...)
	errorStr := string(errBuffer.Bytes())
	outStr := string(outBuffer.Bytes())

	if err != nil {
		if !errorutil.IsExitStatusError(err) {
			return err
		}

		if errorStr != "" && !errorutil.IsExitStatusErrorStr(errorStr) {
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
