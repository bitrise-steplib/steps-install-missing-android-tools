package sdk

import (
	"path/filepath"

	"fmt"

	"github.com/bitrise-tools/go-android/sdkcomponent"
)

// Model ...
type Model struct {
	androidHome  string
	buildTools   []sdkcomponent.BuildTool
	platforms    []sdkcomponent.Platform
	systemImages []sdkcomponent.SystemImage
}

// AndroidSdkInterface ...
type AndroidSdkInterface interface {
	GetAndroidHome() string
}

// New ...
func New(androidHome string) (*Model, error) {
	androidHomeEval, err := filepath.EvalSymlinks(androidHome)
	if err != nil {
		return nil, fmt.Errorf("Failed to evaulate symlink of (%s), error: %s", androidHome, err)
	}
	return &Model{androidHome: androidHomeEval}, nil
}

// GetAndroidHome ...
func (model *Model) GetAndroidHome() string {
	return model.androidHome
}
