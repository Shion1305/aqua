package registry_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/aquaproj/aqua/v2/pkg/config"
	"github.com/aquaproj/aqua/v2/pkg/config/aqua"
	cfgRegistry "github.com/aquaproj/aqua/v2/pkg/config/registry"
	"github.com/aquaproj/aqua/v2/pkg/cosign"
	"github.com/aquaproj/aqua/v2/pkg/download"
	registry "github.com/aquaproj/aqua/v2/pkg/install-registry"
	"github.com/aquaproj/aqua/v2/pkg/runtime"
	"github.com/aquaproj/aqua/v2/pkg/slsa"
	"github.com/aquaproj/aqua/v2/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	"github.com/suzuki-shunsuke/flute/flute"
)

func TestInstaller_InstallHTTPRegistry(t *testing.T) { //nolint:funlen
	t.Parallel()
	logE := logrus.NewEntry(logrus.New())
	data := []struct {
		name        string
		param       *config.Param
		cfg         *aqua.Config
		cfgFilePath string
		isErr       bool
		exp         map[string]*cfgRegistry.Config
		transport   *flute.Transport
	}{
		{
			name: "http registry with direct YAML",
			param: &config.Param{
				MaxParallelism: 5,
				RootDir:        "/root/.aqua",
			},
			cfgFilePath: "aqua.yaml",
			cfg: &aqua.Config{
				Registries: aqua.Registries{
					"http-registry": {
						Type:    "http",
						Name:    "http-registry",
						URL:     "https://example.com/registry/{{.Version}}/registry.yaml",
						Version: "v1.0.0",
					},
				},
			},
			exp: map[string]*cfgRegistry.Config{
				"http-registry": {
					PackageInfos: cfgRegistry.PackageInfos{
						{
							Type:      "github_release",
							RepoOwner: "test-owner",
							RepoName:  "test-repo",
							Asset:     "test-{{.Version}}.tar.gz",
						},
					},
				},
			},
			transport: &flute.Transport{
				Services: []flute.Service{
					{
						Endpoint: "https://example.com",
						Routes: []flute.Route{
							{
								Name: "download http registry",
								Matcher: &flute.Matcher{
									Method: "GET",
									Path:   "/registry/v1.0.0/registry.yaml",
								},
								Response: &flute.Response{
									Base: http.Response{
										StatusCode: http.StatusOK,
									},
									BodyString: `packages:
- type: github_release
  repo_owner: test-owner
  repo_name: test-repo
  asset: "test-{{.Version}}.tar.gz"
`,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "http registry with JSON",
			param: &config.Param{
				MaxParallelism: 5,
				RootDir:        "/root/.aqua",
			},
			cfgFilePath: "aqua.yaml",
			cfg: &aqua.Config{
				Registries: aqua.Registries{
					"http-json": {
						Type:    "http",
						Name:    "http-json",
						URL:     "https://example.com/registry/{{.Version}}/registry.json",
						Version: "v2.0.0",
					},
				},
			},
			exp: map[string]*cfgRegistry.Config{
				"http-json": {
					PackageInfos: cfgRegistry.PackageInfos{
						{
							Type:      "github_release",
							RepoOwner: "json-owner",
							RepoName:  "json-repo",
							Asset:     "json-{{.Version}}.zip",
						},
					},
				},
			},
			transport: &flute.Transport{
				Services: []flute.Service{
					{
						Endpoint: "https://example.com",
						Routes: []flute.Route{
							{
								Name: "download http json registry",
								Matcher: &flute.Matcher{
									Method: "GET",
									Path:   "/registry/v2.0.0/registry.json",
								},
								Response: &flute.Response{
									Base: http.Response{
										StatusCode: http.StatusOK,
									},
									BodyString: `{
  "packages": [
    {
      "type": "github_release",
      "repo_owner": "json-owner",
      "repo_name": "json-repo",
      "asset": "json-{{.Version}}.zip"
    }
  ]
}
`,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "http registry returns 404",
			param: &config.Param{
				MaxParallelism: 5,
				RootDir:        "/root/.aqua",
			},
			cfgFilePath: "aqua.yaml",
			cfg: &aqua.Config{
				Registries: aqua.Registries{
					"http-404": {
						Type:    "http",
						Name:    "http-404",
						URL:     "https://example.com/registry/{{.Version}}/registry.yaml",
						Version: "v1.0.0",
					},
				},
			},
			isErr: true,
			transport: &flute.Transport{
				Services: []flute.Service{
					{
						Endpoint: "https://example.com",
						Routes: []flute.Route{
							{
								Name: "download http registry 404",
								Matcher: &flute.Matcher{
									Method: "GET",
									Path:   "/registry/v1.0.0/registry.yaml",
								},
								Response: &flute.Response{
									Base: http.Response{
										StatusCode: http.StatusNotFound,
									},
									BodyString: "Not Found",
								},
							},
						},
					},
				},
			},
		},
	}
	rt := &runtime.Runtime{
		GOOS:   "linux",
		GOARCH: "amd64",
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			fs, err := testutil.NewFs(map[string]string{})
			if err != nil {
				t.Fatal(err)
			}
			httpDownloader := download.NewHTTPDownloader(logE, &http.Client{
				Transport: d.transport,
			})
			inst := registry.New(d.param, nil, httpDownloader, fs, rt, &cosign.MockVerifier{}, &slsa.MockVerifier{})
			registries, err := inst.InstallRegistries(ctx, logE, d.cfg, d.cfgFilePath, nil)
			if err != nil {
				if d.isErr {
					return
				}
				t.Fatal(err)
			}
			if d.isErr {
				t.Fatal("error must be returned")
			}
			if diff := cmp.Diff(d.exp, registries, cmp.AllowUnexported(cfgRegistry.Config{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
