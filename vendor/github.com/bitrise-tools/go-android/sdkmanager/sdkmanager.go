package sdkmanager

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-android/sdk"
	"github.com/bitrise-tools/go-android/sdkcomponent"
)

// Model ...
type Model struct {
	androidHome string
	legacy      bool
	binPth      string
}

// IsLegacySDKManager ...
func IsLegacySDKManager(androidHome string) (bool, error) {
	exist, err := pathutil.IsPathExists(filepath.Join(androidHome, "tools", "bin", "sdkmanager"))
	return !exist, err
}

// New ...
func New(androidSdkInterface sdk.AndroidSdkInterface) (*Model, error) {
	binPth := filepath.Join(androidSdkInterface.GetAndroidHome(), "tools", "bin", "sdkmanager")

	legacy, err := IsLegacySDKManager(androidSdkInterface.GetAndroidHome())
	if err != nil {
		return nil, err
	} else if legacy {
		binPth = filepath.Join(androidSdkInterface.GetAndroidHome(), "tools", "android")
	}

	if exist, err := pathutil.IsPathExists(binPth); err != nil {
		return nil, err
	} else if !exist {
		return nil, fmt.Errorf("no sdk manager tool found at: %s", binPth)
	}

	return &Model{
		androidHome: androidSdkInterface.GetAndroidHome(),
		legacy:      legacy,
		binPth:      binPth,
	}, nil
}

// IsLegacySDK ...
func (model Model) IsLegacySDK() bool {
	return model.legacy
}

// IsInstalled ...
func (model Model) IsInstalled(component sdkcomponent.Model) (bool, error) {
	relPth := component.InstallPathInAndroidHome()
	installPth := filepath.Join(model.androidHome, relPth)
	return pathutil.IsPathExists(installPth)
}

// InstallCommand ...
func (model Model) InstallCommand(component sdkcomponent.Model) *command.Model {
	if model.legacy {
		return command.New(model.binPth, "update", "sdk", "--no-ui", "--all", "--filter", component.GetLegacySDKStylePath())
	}
	return command.New(model.binPth, component.GetSDKStylePath())
}
