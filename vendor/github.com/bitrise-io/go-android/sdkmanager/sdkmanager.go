package sdkmanager

import (
	"fmt"
	"path/filepath"
	"strings"

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
	cmdFactory  command.Factory
}

// New ...
func New(sdk sdk.AndroidSdkInterface, cmdFactory command.Factory) (*Model, error) {
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
			cmdFactory:  cmdFactory,
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
			cmdFactory:  cmdFactory,
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
func (model Model) InstallCommand(component sdkcomponent.Model) command.Command {
	if model.legacy {
		args := []string{"update", "sdk", "--no-ui", "--all", "--filter", component.GetLegacySDKStylePath()}
		return model.cmdFactory.Create(model.binPth, args, nil)
	}
	cmdOpts := command.Opts{
		Stdin: strings.NewReader("y"), // Accept license if prompted
	}
	return model.cmdFactory.Create(model.binPth, []string{component.GetSDKStylePath()}, &cmdOpts)
}
