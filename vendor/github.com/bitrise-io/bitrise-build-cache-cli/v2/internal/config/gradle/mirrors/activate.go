package mirrors

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

// EnabledEnvKey is the env var that gates mirror activation.
const EnabledEnvKey = "BITRISE_MAVENCENTRAL_PROXY_ENABLED"

// DatacenterEnvKey identifies the Bitrise datacenter the build runs in.
const DatacenterEnvKey = "BITRISE_DEN_VM_DATACENTER"

// Params bundles the inputs needed to write the Gradle mirrors init script.
type Params struct {
	GradleHome string       // absolute path to the Gradle home (e.g. ~/.gradle expanded)
	Mirrors    []RepoMirror // mirrors to install
	Datacenter string       // datacenter (e.g. "AMS1") used to build the mirror URL
	Enabled    bool         // when false, Activate is a no-op
}

type templateEntry struct {
	ID                      string
	GradleMatch             string
	MirrorURL               string
	ApplyToPluginManagement bool
}

type templateData struct {
	Mirrors []templateEntry
}

// Activate writes the Gradle init script when mirror activation is enabled and
// at least one mirror is selected. Otherwise it logs the reason and returns
// nil. The init script is placed at <GradleHome>/init.d/<InitFileName>.
func Activate(logger log.Logger, osProxy utils.OsProxy, params Params) error {
	if !params.Enabled {
		logger.Infof("%s is not set to \"true\", skipping Gradle mirror activation", EnabledEnvKey)

		return nil
	}

	if params.Datacenter == "" {
		logger.Infof("%s is not set, skipping Gradle mirror activation (e.g. local dev environment)", DatacenterEnvKey)

		return nil
	}

	if len(params.Mirrors) == 0 {
		logger.Infof("No mirrors selected, skipping Gradle mirror activation")

		return nil
	}

	region := DatacenterToRegion(params.Datacenter)
	if !IsSupportedRegion(region) {
		logger.Infof("Datacenter %q (region %q) has no Bitrise mirror deployment, skipping Gradle mirror activation", params.Datacenter, region)

		return nil
	}

	entries := make([]templateEntry, 0, len(params.Mirrors))
	for _, m := range params.Mirrors {
		url := fmt.Sprintf(URLPattern, region, m.URLSegment)
		entries = append(entries, templateEntry{
			ID:                      m.TemplateID,
			GradleMatch:             m.GradleMatch,
			MirrorURL:               url,
			ApplyToPluginManagement: m.ApplyToPluginManagement,
		})
		logger.Debugf("Mirror %s: region=%s, URL=%s", m.FlagName, region, url)
	}

	tmpl, err := template.New("gradle-mirrors").Parse(initTemplate)
	if err != nil {
		return fmt.Errorf("parse gradle-mirrors init template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData{Mirrors: entries}); err != nil {
		return fmt.Errorf("execute gradle-mirrors init template: %w", err)
	}

	initDPath := filepath.Join(params.GradleHome, "init.d")
	if err := osProxy.MkdirAll(initDPath, 0o755); err != nil { //nolint:mnd
		return fmt.Errorf("ensure ~/.gradle/init.d exists: %w", err)
	}

	initFilePath := filepath.Join(initDPath, InitFileName)
	logger.Debugf("Writing Gradle mirrors init script to %s", initFilePath)

	if err := osProxy.WriteFile(initFilePath, buf.Bytes(), 0o644); err != nil { //nolint:gosec,mnd
		return fmt.Errorf("write %s: %w", initFilePath, err)
	}

	logger.Infof("Gradle mirrors activated")

	return nil
}
