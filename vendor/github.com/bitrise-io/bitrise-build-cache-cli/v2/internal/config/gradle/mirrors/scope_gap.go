package mirrors

import (
	"path/filepath"
	"regexp"

	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

// gradleScriptCandidates are the script files scanned at activation time.
// We check the project root + the conventional Android `app/` module dir.
// Anything declared inside `apply(from = ...)`-loaded scripts' own
// buildscript {} blocks is invisible to the init script — those scripts'
// ScriptHandler instances are not reachable from gradle.beforeSettings or
// gradle.beforeProject hooks. Surfacing the file makes the gap actionable.
//
//nolint:gochecknoglobals
var gradleScriptCandidates = []string{
	"settings.gradle",
	"settings.gradle.kts",
	"build.gradle",
	"build.gradle.kts",
	filepath.Join("app", "build.gradle"),
	filepath.Join("app", "build.gradle.kts"),
}

//nolint:gochecknoglobals
var (
	applyFromKotlinRe = regexp.MustCompile(`apply\s*\(\s*from\s*=`)
	applyFromGroovyRe = regexp.MustCompile(`\bapply\s+from\s*:`)
)

// LogScopeGapWarnings scans the project for Gradle scripts that pull in other
// scripts via `apply(from = ...)`. Repositories declared inside those applied
// scripts' own buildscript {} blocks are not redirected by the mirror init
// script, so dependencies they resolve still hit the upstream repo and may
// fail with rate limits or verification mismatches.
func LogScopeGapWarnings(logger log.Logger, osProxy utils.OsProxy, projectRoot string) {
	hits := scanForApplyFrom(osProxy, projectRoot)
	if len(hits) == 0 {
		return
	}

	logger.Warnf("Detected %d Gradle script(s) using `apply(from = ...)`:", len(hits))

	for _, h := range hits {
		logger.Warnf("  - %s", h)
	}

	logger.Warnf("Bitrise repository mirrors do NOT cover repositories declared inside applied scripts' own buildscript {} blocks.")
	logger.Warnf("If those scripts declare e.g. mavenCentral() in a buildscript {} block, dependencies fetched from there go to the upstream repo and may hit rate limits or verification failures.")
	logger.Warnf("Recommended: move plugin classpath into settings.gradle.kts pluginManagement {}, which the mirror covers.")
}

func scanForApplyFrom(osProxy utils.OsProxy, projectRoot string) []string {
	var hits []string

	for _, name := range gradleScriptCandidates {
		path := filepath.Join(projectRoot, name)

		content, ok, err := osProxy.ReadFileIfExists(path)
		if err != nil || !ok {
			continue
		}

		if applyFromKotlinRe.MatchString(content) || applyFromGroovyRe.MatchString(content) {
			hits = append(hits, path)
		}
	}

	return hits
}
