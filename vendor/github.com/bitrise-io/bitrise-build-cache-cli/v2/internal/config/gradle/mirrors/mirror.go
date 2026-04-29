// Package mirrors provides the data model and activation logic for installing
// Bitrise repository mirrors as a Gradle init script.
package mirrors

import (
	_ "embed"
	"strings"
	"unicode"
)

// RepoMirror describes a single repository that can be mirrored.
type RepoMirror struct {
	FlagName                string // cobra flag name, e.g. "mavencentral"
	TemplateID              string // unique suffix for Kotlin variable names, e.g. "Central"
	URLSegment              string // last path segment in the mirror URL, e.g. "central"
	GradleMatch             string // Kotlin predicate body (using `r` as the repo) that decides whether the repo should be mirrored
	ApplyToPluginManagement bool   // also apply this mirror to pluginManagement.repositories
}

// KnownMirrors is the registry of supported mirrors.
// Order matters: entries are applied in the listed order, so URL-based predicates
// (e.g. apache-central) must run before name-based ones that overwrite the URL.
var KnownMirrors = []RepoMirror{ //nolint:gochecknoglobals
	{FlagName: "mavencentral-apache", TemplateID: "ApacheCentral", URLSegment: "apache-central", GradleMatch: `r.getUrl().toString().trimEnd('/').equals("https://repo.maven.apache.org/maven2")`, ApplyToPluginManagement: true},
	{FlagName: "mavencentral", TemplateID: "Central", URLSegment: "central", GradleMatch: `r.getName().equals(ArtifactRepositoryContainer.DEFAULT_MAVEN_CENTRAL_REPO_NAME) || r.getUrl().toString().trimEnd('/') in setOf("https://repo1.maven.org/maven2", "https://jcenter.bintray.com")`},
	{FlagName: "google", TemplateID: "Google", URLSegment: "google", GradleMatch: `r.getName().equals("Google")`},
}

//go:embed asset/gradle-mirrors.init.gradle.kts.gotemplate
var initTemplate string

// InitTemplate returns the embedded Gradle init script template source.
func InitTemplate() string {
	return initTemplate
}

// InitFileName is the filename written under ~/.gradle/init.d.
const InitFileName = "bitrise-gradle-mirrors.init.gradle.kts"

// URLPattern is the format string for mirror URLs: region + URL segment.
const URLPattern = "https://repository-manager-%s.services.bitrise.io:8090/maven/%s"

// FilterByFlagNames returns the subset of KnownMirrors whose FlagName matches
// any of names. If names is empty, KnownMirrors is returned unchanged. Order
// follows KnownMirrors.
func FilterByFlagNames(names []string) []RepoMirror {
	if len(names) == 0 {
		return KnownMirrors
	}

	wanted := make(map[string]struct{}, len(names))
	for _, n := range names {
		wanted[n] = struct{}{}
	}

	selected := make([]RepoMirror, 0, len(names))

	for _, m := range KnownMirrors {
		if _, ok := wanted[m.FlagName]; ok {
			selected = append(selected, m)
		}
	}

	return selected
}

// DatacenterToRegion converts a datacenter env value (e.g. "AMS1", "IAD1", "ORD1")
// to the mirror region slug by lowercasing and stripping trailing digits.
func DatacenterToRegion(dc string) string {
	lower := strings.ToLower(dc)

	return strings.TrimRightFunc(lower, unicode.IsDigit)
}

// SupportedRegions lists the datacenter region slugs that have a Bitrise mirror
// deployed. Datacenters outside this set are skipped during activation.
var SupportedRegions = []string{"iad", "ord", "ams"} //nolint:gochecknoglobals

// IsSupportedRegion reports whether the given region slug (e.g. "iad") has a
// Bitrise mirror deployment.
func IsSupportedRegion(region string) bool {
	for _, r := range SupportedRegions {
		if r == region {
			return true
		}
	}

	return false
}
