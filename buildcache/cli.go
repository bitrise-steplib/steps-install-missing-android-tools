package buildcache

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	cliVersion       = "v1.4.0-alpha.3"
	installerURL     = "https://raw.githubusercontent.com/bitrise-io/bitrise-build-cache-cli/main/install/installer.sh"
	artifactRegistry = "https://artifactregistry.googleapis.com/download/v1/projects/ip-build-cache-prod/locations/us-central1/repositories/build-cache-cli-releases/files"
)

// DownloadAndActivateMavenCentralMirror downloads the bitrise-build-cache CLI
// and runs `activate mavencentral-mirror`. The CLI itself noops when
// BITRISE_MAVENCENTRAL_PROXY_ENABLED is not "true".
func DownloadAndActivateMavenCentralMirror() error {
	binDir := filepath.Join(os.TempDir(), "bitrise-build-cache-bin")
	binaryPath := filepath.Join(binDir, "bitrise-build-cache")

	if err := downloadCLI(binDir, binaryPath); err != nil {
		return fmt.Errorf("download bitrise-build-cache CLI: %w", err)
	}

	return runActivateMavenCentralMirror(binaryPath)
}

func downloadCLI(binDir, binaryPath string) error {
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("create bin directory: %w", err)
	}

	// Try GitHub installer first
	if err := downloadViaInstaller(binDir); err == nil {
		if _, err := os.Stat(binaryPath); err == nil {
			return nil
		}
	}

	// Fall back to Artifact Registry
	if err := downloadFromArtifactRegistry(binDir, binaryPath); err != nil {
		return fmt.Errorf("artifact registry fallback: %w", err)
	}

	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("CLI binary not found after download attempts")
	}
	return nil
}

func downloadViaInstaller(binDir string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(installerURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("installer script returned HTTP %d", resp.StatusCode)
	}

	installerScript, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	cmd := exec.Command("sh", "-s", "--", "-b", binDir, "-d", cliVersion)
	cmd.Stdin = strings.NewReader(string(installerScript))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}

	return nil
}

func downloadFromArtifactRegistry(binDir, binaryPath string) error {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	version := strings.TrimPrefix(cliVersion, "v")
	packageName := fmt.Sprintf("bitrise-build-cache_%s_%s.tar.gz", osName, arch)
	filename := fmt.Sprintf("bitrise-build-cache_%s_%s_%s.tar.gz", version, osName, arch)
	filePath := fmt.Sprintf("%s:%s:%s", packageName, version, filename)

	url := fmt.Sprintf("%s/%s:download?alt=media", artifactRegistry, filePath)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("artifact registry returned HTTP %d", resp.StatusCode)
	}

	// Extract tar.gz directly from response body
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		if header.Name == "bitrise-build-cache" || filepath.Base(header.Name) == "bitrise-build-cache" {
			outFile, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("create binary file: %w", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("write binary: %w", err)
			}
			outFile.Close()
			return nil
		}
	}

	return fmt.Errorf("bitrise-build-cache binary not found in tar archive")
}

func runActivateMavenCentralMirror(binaryPath string) error {
	cmd := exec.Command(binaryPath, "activate", "mavencentral-mirror")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("activate mavencentral-mirror: %w", err)
	}

	return nil
}
