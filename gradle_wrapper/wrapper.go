package gradle_wrapper

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/log"
)

func FindAll(dir string) ([]string, error) {
	var wrappers []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("failed to walk path: %s", err)
			return nil
		}

		if info.IsDir() {
			// Skip certain directories that are commonly found in projects and are unlikely to contain gradle wrappers
			if info.Name() == ".git" || info.Name() == "build" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		name := info.Name()
		if strings.EqualFold(name, "gradlew") || strings.EqualFold(name, "gradlew.bat") {
			wrappers = append(wrappers, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return wrappers, nil
}
