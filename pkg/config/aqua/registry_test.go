package aqua_test

import (
	"testing"

	"github.com/aquaproj/aqua/v2/pkg/config/aqua"
)

func TestRegistry_Validate(t *testing.T) { //nolint:funlen
	t.Parallel()
	data := []struct {
		title    string
		registry *aqua.Registry
		isErr    bool
	}{
		{
			title: "github_content",
			registry: &aqua.Registry{
				RepoOwner: "aquaproj",
				RepoName:  "aqua-registry",
				Ref:       "v0.8.0",
				Path:      "foo.yaml",
				Type:      "github_content",
			},
		},
		{
			title: "github_content repo_owner is required",
			registry: &aqua.Registry{
				RepoName: "aqua-registry",
				Ref:      "v0.8.0",
				Path:     "foo.yaml",
				Type:     "github_content",
			},
			isErr: true,
		},
		{
			title: "github_content repo_name is required",
			registry: &aqua.Registry{
				RepoOwner: "aquaproj",
				Ref:       "v0.8.0",
				Path:      "foo.yaml",
				Type:      "github_content",
			},
			isErr: true,
		},
		{
			title: "github_content ref is required",
			registry: &aqua.Registry{
				RepoOwner: "aquaproj",
				RepoName:  "aqua-registry",
				Path:      "foo.yaml",
				Type:      "github_content",
			},
			isErr: true,
		},
		{
			title: "local",
			registry: &aqua.Registry{
				Path: "foo.yaml",
				Type: "local",
			},
		},
		{
			title: "local path is required",
			registry: &aqua.Registry{
				Type: "local",
			},
			isErr: true,
		},
		{
			title: "invalid type",
			registry: &aqua.Registry{
				Type: "invalid-type",
			},
			isErr: true,
		},
		{
			title: "http",
			registry: &aqua.Registry{
				Type:    "http",
				URL:     "https://example.com/registry/{{.Version}}/registry.yaml",
				Version: "v1.0.0",
			},
		},
		{
			title: "http url is required",
			registry: &aqua.Registry{
				Type:    "http",
				Version: "v1.0.0",
			},
			isErr: true,
		},
		{
			title: "http version is required",
			registry: &aqua.Registry{
				Type: "http",
				URL:  "https://example.com/registry/{{.Version}}/registry.yaml",
			},
			isErr: true,
		},
		{
			title: "http url must contain {{.Version}}",
			registry: &aqua.Registry{
				Type:    "http",
				URL:     "https://example.com/registry/v1.0.0/registry.yaml",
				Version: "v1.0.0",
			},
			isErr: true,
		},
	}
	for _, d := range data {
		t.Run(d.title, func(t *testing.T) {
			t.Parallel()
			if err := d.registry.Validate(); err != nil {
				if d.isErr {
					return
				}
				t.Fatal(err)
			}
			if d.isErr {
				t.Fatal("error must be returned")
			}
		})
	}
}

func TestRegistry_FilePath(t *testing.T) {
	t.Parallel()
	data := []struct {
		title       string
		exp         string
		registry    *aqua.Registry
		rootDir     string
		homeDir     string
		cfgFilePath string
		isErr       bool
	}{
		{
			title:       "normal",
			exp:         "ci/foo.yaml",
			rootDir:     "/root/.aqua",
			homeDir:     "/root",
			cfgFilePath: "ci/aqua.yaml",
			registry: &aqua.Registry{
				Path: "foo.yaml",
				Type: "local",
			},
		},
		{
			title:   "github_content",
			exp:     "/root/.aqua/registries/github_content/github.com/aquaproj/aqua-registry/v0.8.0/foo.yaml",
			rootDir: "/root/.aqua",
			registry: &aqua.Registry{
				RepoOwner: "aquaproj",
				RepoName:  "aqua-registry",
				Ref:       "v0.8.0",
				Path:      "foo.yaml",
				Type:      "github_content",
			},
		},
		{
			title:   "http with path",
			exp:     "/root/.aqua/registries/http/06eeabea3ca08429/v1.0.0/custom.yaml",
			rootDir: "/root/.aqua",
			registry: &aqua.Registry{
				Type:    "http",
				URL:     "https://example.com/registry/{{.Version}}/registry.tar.gz",
				Version: "v1.0.0",
				Path:    "custom.yaml",
			},
		},
		{
			title:   "http without path",
			exp:     "/root/.aqua/registries/http/06eeabea3ca08429/v1.2.3/registry.yaml",
			rootDir: "/root/.aqua",
			registry: &aqua.Registry{
				Type:    "http",
				URL:     "https://example.com/registry/{{.Version}}/registry.tar.gz",
				Version: "v1.2.3",
			},
		},
	}
	for _, d := range data {
		t.Run(d.title, func(t *testing.T) {
			t.Parallel()
			p, err := d.registry.FilePath(d.rootDir, d.cfgFilePath)
			if err != nil {
				if d.isErr {
					return
				}
				t.Fatal(err)
			}
			if d.isErr {
				t.Fatal("error must be returned")
			}
			if p != d.exp {
				t.Fatalf("wanted %s, got %s", d.exp, p)
			}
		})
	}
}
