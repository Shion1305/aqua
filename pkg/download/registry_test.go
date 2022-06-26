package download_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/clivm/clivm/pkg/download"
	githubSvc "github.com/clivm/clivm/pkg/github"
	"github.com/clivm/clivm/pkg/runtime"
	"github.com/google/go-github/v44/github"
	"github.com/sirupsen/logrus"
	"github.com/suzuki-shunsuke/flute/flute"
)

func Test_registryDownloader_GetGitHubContentFile(t *testing.T) { //nolint:funlen
	t.Parallel()
	data := []struct {
		name       string
		repoOwner  string
		repoName   string
		ref        string
		path       string
		rt         *runtime.Runtime
		isErr      bool
		exp        string
		github     githubSvc.RepositoryService
		httpClient *http.Client
	}{
		{
			name:      "github_content http",
			repoOwner: "clivm",
			repoName:  "aqua-registry",
			ref:       "v2.16.0",
			path:      "registry.yaml",
			exp:       "foo",
			github:    nil,
			httpClient: &http.Client{
				Transport: &flute.Transport{
					Services: []flute.Service{
						{
							Endpoint: "https://raw.githubusercontent.com",
							Routes: []flute.Route{
								{
									Name: "download an asset",
									Matcher: &flute.Matcher{
										Method: "GET",
										Path:   "/clivm/clivm-registry/v2.16.0/registry.yaml",
									},
									Response: &flute.Response{
										Base: http.Response{
											StatusCode: 200,
										},
										BodyString: "foo",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:      "github_content github api",
			repoOwner: "clivm",
			repoName:  "aqua-registry",
			ref:       "v2.16.0",
			path:      "registry.yaml",
			exp:       "foo",
			github: &githubSvc.MockRepositoryService{
				Content: &github.RepositoryContent{
					Content: stringP("foo"),
				},
			},
			httpClient: &http.Client{
				Transport: &flute.Transport{
					Services: []flute.Service{
						{
							Endpoint: "https://raw.githubusercontent.com",
							Routes: []flute.Route{
								{
									Name: "download an asset",
									Matcher: &flute.Matcher{
										Method: "GET",
										Path:   "/clivm/clivm-registry/v2.16.0/registry.yaml",
									},
									Response: &flute.Response{
										Base: http.Response{
											StatusCode: 400,
										},
										BodyString: "invalid request",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	logE := logrus.NewEntry(logrus.New())
	ctx := context.Background()
	for _, d := range data {
		d := d
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			downloader := download.NewRegistryDownloader(d.github, download.NewHTTPDownloader(d.httpClient))
			file, err := downloader.GetGitHubContentFile(ctx, d.repoOwner, d.repoName, d.ref, d.path, logE)
			if err != nil {
				if d.isErr {
					return
				}
				t.Fatal(err)
			}
			if d.isErr {
				t.Fatal("error must be returned")
			}
			if string(file) != d.exp {
				t.Fatalf("wanted %s, got %s", d.exp, string(file))
			}
		})
	}
}
