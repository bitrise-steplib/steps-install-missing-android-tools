package sdk

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-utils/pathutil"
)

// Model ...
type Model struct {
	androidHome string
}

// AndroidSdkInterface ...
type AndroidSdkInterface interface {
	GetAndroidHome() string
}

// New ...
func New(androidHome string) (*Model, error) {
	evaluatedAndroidHome, err := filepath.EvalSymlinks(androidHome)
	if err != nil {
		return nil, err
	}

	if exist, err := pathutil.IsDirExists(evaluatedAndroidHome); err != nil {
		return nil, err
	} else if !exist {
		return nil, fmt.Errorf("android home not exists at: %s", evaluatedAndroidHome)
	}

	return &Model{androidHome: evaluatedAndroidHome}, nil
}

// GetAndroidHome ...
func (model *Model) GetAndroidHome() string {
	return model.androidHome
}
