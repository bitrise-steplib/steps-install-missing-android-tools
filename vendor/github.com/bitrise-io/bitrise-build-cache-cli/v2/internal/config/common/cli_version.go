package common

import (
	"fmt"
	"os"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
)

func GetCLIVersion(logger log.Logger) string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		logger.Infof("Failed to read build info")

		return "unknown"
	}

	// Find the bitrise-build-cache-cli module in the build info.
	// If ran from source, it will be the main module.
	// If ran from a step, it will be a dependency module.
	modules := []*debug.Module{&bi.Main}
	modules = append(modules, bi.Deps...)
	idx := slices.IndexFunc(modules, func(c *debug.Module) bool { return strings.Contains(c.Path, "bitrise-build-cache-cli") })
	if idx == -1 || idx >= len(modules) {
		logger.Infof("Failed to find bitrise-build-cache-cli module in build info")

		return "unknown"
	}

	return modules[idx].Version
}

// LogCLIVersion writes a single line with the resolved CLI version to STDERR.
// Stderr (not stdout) is intentional: some callers — e.g. xcodebuild wrappers
// fronted by `@react-native-community/cli-platform-apple` — parse the CLI's
// stdout as JSON. Writing the version line to stdout breaks that JSON parse.
// Call this from public entry points (cobra root PersistentPreRun, pkg/*
// Activator/Runner methods) so each invocation records which CLI the caller
// is running.
func LogCLIVersion(logger log.Logger) {
	_, _ = fmt.Fprintf(os.Stderr, "Bitrise Build Cache CLI version: %s\n", GetCLIVersion(logger))
}
