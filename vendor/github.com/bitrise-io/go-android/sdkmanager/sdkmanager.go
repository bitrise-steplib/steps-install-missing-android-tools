package sdkmanager

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-android/sdk"
	"github.com/bitrise-io/go-android/sdkcomponent"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Model ...
type Model struct {
	androidHome string
	legacy      bool
	binPth      string
}

// New ...
func New(sdk sdk.AndroidSdkInterface) (*Model, error) {
	cmdlineToolsPath, err := sdk.CmdlineToolsPath()
	if err != nil {
		return nil, err
	}

	sdkmanagerPath := filepath.Join(cmdlineToolsPath, "sdkmanager")
	if exist, err := pathutil.IsPathExists(sdkmanagerPath); err != nil {
		return nil, err
	} else if exist {
		return &Model{
			androidHome: sdk.GetAndroidHome(),
			binPth:      sdkmanagerPath,
		}, nil
	}

	legacySdkmanagerPath := filepath.Join(cmdlineToolsPath, "android")
	if exist, err := pathutil.IsPathExists(legacySdkmanagerPath); err != nil {
		return nil, err
	} else if exist {
		return &Model{
			androidHome: sdk.GetAndroidHome(),
			legacy:      true,
			binPth:      legacySdkmanagerPath,
		}, nil
	}

	return nil, fmt.Errorf("no sdkmanager tool found at: %s", sdkmanagerPath)
}

// IsLegacySDK ...
func (model Model) IsLegacySDK() bool {
	return model.legacy
}

// IsInstalled ...
func (model Model) IsInstalled(component sdkcomponent.Model) (bool, error) {
	relPth := component.InstallPathInAndroidHome()
	indicatorFile := component.InstallationIndicatorFile()
	installPth := filepath.Join(model.androidHome, relPth)

	if indicatorFile != "" {
		installPth = filepath.Join(installPth, indicatorFile)
	}
	return pathutil.IsPathExists(installPth)
}

// InstallCommand ...
func (model Model) InstallCommand(component sdkcomponent.Model) *command.Model {
	if model.legacy {
		return command.New(model.binPth, "update", "sdk", "--no-ui", "--all", "--filter", component.GetLegacySDKStylePath())
	}
	return command.New(model.binPth, component.GetSDKStylePath())
}
