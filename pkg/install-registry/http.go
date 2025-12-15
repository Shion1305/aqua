package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/aquaproj/aqua/v2/pkg/checksum"
	"github.com/aquaproj/aqua/v2/pkg/config/aqua"
	"github.com/aquaproj/aqua/v2/pkg/config/registry"
	"github.com/aquaproj/aqua/v2/pkg/template"
	"github.com/aquaproj/aqua/v2/pkg/unarchive"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v2"
)

// getHTTPRegistry downloads and installs an HTTP registry file.
func (is *Installer) getHTTPRegistry(ctx context.Context, logE *logrus.Entry, regist *aqua.Registry, registryFilePath string, checksums *checksum.Checksums) (*registry.Config, error) {
	// Render the URL with the version
	renderedURL, err := template.Execute(regist.URL, map[string]any{
		"Version": regist.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("render registry URL: %w", err)
	}

	logE = logE.WithFields(logrus.Fields{
		"registry_name": regist.Name,
		"registry_url":  renderedURL,
		"version":       regist.Version,
	})
	logE.Debug("downloading HTTP registry")

	// Download the file
	body, _, err := is.httpDownloader.Download(ctx, renderedURL)
	if err != nil {
		return nil, fmt.Errorf("download registry from HTTP: %w", err)
	}
	defer body.Close()

	// Read the content
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read registry content: %w", err)
	}

	// Verify checksum if provided
	if checksums != nil {
		if err := checksum.CheckRegistry(regist, checksums, content); err != nil {
			return nil, fmt.Errorf("check a registry's checksum: %w", err)
		}
	}

	// Create parent directory
	if err := is.fs.MkdirAll(filepath.Dir(registryFilePath), 0o755); err != nil {
		return nil, fmt.Errorf("create the parent directory of the registry file: %w", err)
	}

	// If format is specified, it's an archive that needs extraction
	if regist.Format != "" {
		return is.extractHTTPRegistryArchive(ctx, logE, regist, registryFilePath, content)
	}

	// Otherwise, treat it as a direct YAML/JSON file
	return is.saveHTTPRegistry(regist, registryFilePath, content)
}

// saveHTTPRegistry saves the registry content to disk and parses it.
func (is *Installer) saveHTTPRegistry(regist *aqua.Registry, registryFilePath string, content []byte) (*registry.Config, error) {
	// Write the file
	if err := afero.WriteFile(is.fs, registryFilePath, content, registryFilePermission); err != nil {
		return nil, fmt.Errorf("write the registry file: %w", err)
	}

	// Parse the content
	registryContent := &registry.Config{}
	if isJSON(registryFilePath) {
		if err := json.Unmarshal(content, registryContent); err != nil {
			return nil, fmt.Errorf("parse the registry configuration file as JSON: %w", err)
		}
		return registryContent, nil
	}
	if err := yaml.Unmarshal(content, registryContent); err != nil {
		return nil, fmt.Errorf("parse the registry configuration file as YAML: %w", err)
	}
	return registryContent, nil
}

// extractHTTPRegistryArchive extracts a registry from an archive.
func (is *Installer) extractHTTPRegistryArchive(ctx context.Context, logE *logrus.Entry, regist *aqua.Registry, registryFilePath string, content []byte) (*registry.Config, error) {
	// Create a temporary file for the archive
	tempDir := filepath.Join(filepath.Dir(registryFilePath), ".tmp")
	if err := is.fs.MkdirAll(tempDir, 0o755); err != nil {
		return nil, fmt.Errorf("create temporary directory: %w", err)
	}
	defer is.fs.RemoveAll(tempDir) //nolint:errcheck

	// Determine archive filename based on format
	archiveExt := regist.Format
	if archiveExt == "tar" {
		archiveExt = "tar"
	}
	tempArchivePath := filepath.Join(tempDir, "registry."+archiveExt)

	// Write archive to temporary file
	if err := afero.WriteFile(is.fs, tempArchivePath, content, registryFilePermission); err != nil {
		return nil, fmt.Errorf("write temporary archive file: %w", err)
	}

	// Extract the archive
	extractDir := filepath.Join(tempDir, "extract")
	if err := is.fs.MkdirAll(extractDir, 0o755); err != nil {
		return nil, fmt.Errorf("create extraction directory: %w", err)
	}

	unarchiver := unarchive.New(nil, is.fs)
	if err := unarchiver.Unarchive(ctx, logE, &unarchive.File{
		Body:     &tempFile{path: tempArchivePath, fs: is.fs},
		Filename: tempArchivePath,
		Type:     regist.Format,
	}, extractDir); err != nil {
		return nil, fmt.Errorf("unarchive registry: %w", err)
	}

	// Find the registry file in the extracted content
	registryFileName := "registry.yaml"
	if regist.Path != "" {
		registryFileName = filepath.Base(regist.Path)
	}

	// Try common registry file names if path not specified
	searchPaths := []string{registryFileName}
	if regist.Path == "" {
		searchPaths = append(searchPaths, "registry.yml")
	} else {
		// If path is specified, use the full path
		searchPaths = []string{regist.Path}
	}

	var extractedContent []byte
	var foundPath string
	for _, searchPath := range searchPaths {
		fullPath := filepath.Join(extractDir, searchPath)
		data, err := afero.ReadFile(is.fs, fullPath)
		if err == nil {
			extractedContent = data
			foundPath = fullPath
			break
		}
	}

	if extractedContent == nil {
		return nil, fmt.Errorf("registry file not found in archive (searched: %v)", searchPaths)
	}

	logE.WithField("found_path", foundPath).Debug("found registry file in archive")

	// Save and parse the extracted registry
	return is.saveHTTPRegistry(regist, registryFilePath, extractedContent)
}

// tempFile implements the DownloadedFile interface for archive extraction.
type tempFile struct {
	path string
	fs   afero.Fs
}

func (t *tempFile) Path() (string, error) {
	return t.path, nil
}

func (t *tempFile) ReadLast() (io.ReadCloser, error) {
	return t.fs.Open(t.path) //nolint:wrapcheck
}

func (t *tempFile) Wrap(w io.Writer) io.Writer {
	return w
}
