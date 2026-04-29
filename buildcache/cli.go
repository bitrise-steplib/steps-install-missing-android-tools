package buildcache

import (
	"context"
	"fmt"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/pkg/gradle/mirrors"
)

// ActivateRepoMirrors activates Bitrise repo mirrors for Gradle by calling
// the bitrise-build-cache CLI's gradle-mirrors Activator directly.
func ActivateRepoMirrors(ctx context.Context) error {
	activator := mirrors.NewActivator(mirrors.ActivatorParams{})
	if err := activator.Activate(ctx); err != nil {
		return fmt.Errorf("activate gradle mirrors: %w", err)
	}

	return nil
}
