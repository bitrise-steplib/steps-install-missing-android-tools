package sdk

import "path/filepath"

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
	androidHomeEval, err := filepath.EvalSymlinks(androidHome)
	if err != nil {
		return nil, err
	}
	return &Model{androidHome: androidHomeEval}, nil
}

// GetAndroidHome ...
func (model *Model) GetAndroidHome() string {
	return model.androidHome
}
