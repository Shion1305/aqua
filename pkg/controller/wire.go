//go:build wireinject
// +build wireinject

package controller

import (
	"context"
	"net/http"

	"github.com/aquaproj/aqua/pkg/checksum"
	"github.com/aquaproj/aqua/pkg/config"
	finder "github.com/aquaproj/aqua/pkg/config-finder"
	reader "github.com/aquaproj/aqua/pkg/config-reader"
	"github.com/aquaproj/aqua/pkg/controller/cp"
	cexec "github.com/aquaproj/aqua/pkg/controller/exec"
	"github.com/aquaproj/aqua/pkg/controller/generate"
	genrgst "github.com/aquaproj/aqua/pkg/controller/generate-registry"
	"github.com/aquaproj/aqua/pkg/controller/initcmd"
	"github.com/aquaproj/aqua/pkg/controller/initpolicy"
	"github.com/aquaproj/aqua/pkg/controller/install"
	"github.com/aquaproj/aqua/pkg/controller/list"
	"github.com/aquaproj/aqua/pkg/controller/updateaqua"
	"github.com/aquaproj/aqua/pkg/controller/updatechecksum"
	"github.com/aquaproj/aqua/pkg/controller/which"
	"github.com/aquaproj/aqua/pkg/domain"
	"github.com/aquaproj/aqua/pkg/download"
	"github.com/aquaproj/aqua/pkg/exec"
	"github.com/aquaproj/aqua/pkg/github"
	registry "github.com/aquaproj/aqua/pkg/install-registry"
	"github.com/aquaproj/aqua/pkg/installpackage"
	"github.com/aquaproj/aqua/pkg/link"
	"github.com/aquaproj/aqua/pkg/policy"
	"github.com/aquaproj/aqua/pkg/runtime"
	"github.com/aquaproj/aqua/pkg/unarchive"
	"github.com/google/wire"
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/go-osenv/osenv"
)

func InitializeListCommandController(ctx context.Context, param *config.Param, httpClient *http.Client) *list.Controller {
	wire.Build(
		list.NewController,
		wire.NewSet(
			finder.NewConfigFinder,
			wire.Bind(new(list.ConfigFinder), new(*finder.ConfigFinder)),
		),
		wire.NewSet(
			github.New,
			wire.Bind(new(domain.RepositoriesService), new(*github.RepositoriesService)),
			wire.Bind(new(download.GitHubContentAPI), new(*github.RepositoriesService)),
		),
		wire.NewSet(
			registry.New,
			wire.Bind(new(domain.RegistryInstaller), new(*registry.Installer)),
		),
		wire.NewSet(
			download.NewGitHubContentFileDownloader,
			wire.Bind(new(domain.GitHubContentFileDownloader), new(*download.GitHubContentFileDownloader)),
		),
		wire.NewSet(
			reader.New,
			wire.Bind(new(domain.ConfigReader), new(*reader.ConfigReader)),
		),
		afero.NewOsFs,
		download.NewHTTPDownloader,
	)
	return &list.Controller{}
}

func InitializeGenerateRegistryCommandController(ctx context.Context, param *config.Param, httpClient *http.Client) *genrgst.Controller {
	wire.Build(
		genrgst.NewController,
		wire.NewSet(
			github.New,
			wire.Bind(new(genrgst.RepositoriesService), new(*github.RepositoriesService)),
		),
		afero.NewOsFs,
	)
	return &genrgst.Controller{}
}

func InitializeInitCommandController(ctx context.Context, param *config.Param) *initcmd.Controller {
	wire.Build(
		initcmd.New,
		wire.NewSet(
			github.New,
			wire.Bind(new(initcmd.RepositoriesService), new(*github.RepositoriesService)),
		),
		afero.NewOsFs,
	)
	return &initcmd.Controller{}
}

func InitializeInitPolicyCommandController(ctx context.Context) *initpolicy.Controller {
	wire.Build(
		initpolicy.New,
		afero.NewOsFs,
	)
	return &initpolicy.Controller{}
}

func InitializeGenerateCommandController(ctx context.Context, param *config.Param, httpClient *http.Client) *generate.Controller {
	wire.Build(
		generate.New,
		wire.NewSet(
			finder.NewConfigFinder,
			wire.Bind(new(generate.ConfigFinder), new(*finder.ConfigFinder)),
		),
		wire.NewSet(
			github.New,
			wire.Bind(new(generate.RepositoriesService), new(*github.RepositoriesService)),
			wire.Bind(new(download.GitHubContentAPI), new(*github.RepositoriesService)),
		),
		wire.NewSet(
			registry.New,
			wire.Bind(new(domain.RegistryInstaller), new(*registry.Installer)),
		),
		wire.NewSet(
			download.NewGitHubContentFileDownloader,
			wire.Bind(new(domain.GitHubContentFileDownloader), new(*download.GitHubContentFileDownloader)),
		),
		wire.NewSet(
			reader.New,
			wire.Bind(new(domain.ConfigReader), new(*reader.ConfigReader)),
		),
		afero.NewOsFs,
		generate.NewFuzzyFinder,
		generate.NewVersionSelector,
		download.NewHTTPDownloader,
	)
	return &generate.Controller{}
}

func InitializeInstallCommandController(ctx context.Context, param *config.Param, httpClient *http.Client, rt *runtime.Runtime) *install.Controller {
	wire.Build(
		install.New,
		wire.NewSet(
			finder.NewConfigFinder,
			wire.Bind(new(install.ConfigFinder), new(*finder.ConfigFinder)),
		),
		wire.NewSet(
			github.New,
			wire.Bind(new(domain.RepositoriesService), new(*github.RepositoriesService)),
			wire.Bind(new(download.GitHubContentAPI), new(*github.RepositoriesService)),
		),
		wire.NewSet(
			registry.New,
			wire.Bind(new(domain.RegistryInstaller), new(*registry.Installer)),
		),
		wire.NewSet(
			download.NewGitHubContentFileDownloader,
			wire.Bind(new(domain.GitHubContentFileDownloader), new(*download.GitHubContentFileDownloader)),
		),
		wire.NewSet(
			reader.New,
			wire.Bind(new(domain.ConfigReader), new(*reader.ConfigReader)),
		),
		wire.NewSet(
			installpackage.New,
			wire.Bind(new(domain.PackageInstaller), new(*installpackage.Installer)),
		),
		wire.NewSet(
			download.NewPackageDownloader,
			wire.Bind(new(domain.PackageDownloader), new(*download.PackageDownloader)),
		),
		afero.NewOsFs,
		wire.NewSet(
			link.New,
			wire.Bind(new(domain.Linker), new(*link.Linker)),
		),
		download.NewHTTPDownloader,
		wire.NewSet(
			exec.New,
			wire.Bind(new(installpackage.Executor), new(*exec.Executor)),
		),
		wire.NewSet(
			download.NewChecksumDownloader,
			wire.Bind(new(domain.ChecksumDownloader), new(*download.ChecksumDownloader)),
		),
		wire.NewSet(
			checksum.NewCalculator,
			wire.Bind(new(installpackage.ChecksumCalculator), new(*checksum.Calculator)),
		),
		wire.NewSet(
			unarchive.New,
			wire.Bind(new(installpackage.Unarchiver), new(*unarchive.Unarchiver)),
		),
		wire.NewSet(
			policy.NewChecker,
			wire.Bind(new(domain.PolicyChecker), new(*policy.Checker)),
		),
		wire.NewSet(
			policy.NewConfigReader,
			wire.Bind(new(domain.PolicyConfigReader), new(*policy.ConfigReader)),
		),
	)
	return &install.Controller{}
}

func InitializeWhichCommandController(ctx context.Context, param *config.Param, httpClient *http.Client, rt *runtime.Runtime) *which.Controller {
	wire.Build(
		which.New,
		wire.NewSet(
			finder.NewConfigFinder,
			wire.Bind(new(which.ConfigFinder), new(*finder.ConfigFinder)),
		),
		wire.NewSet(
			github.New,
			wire.Bind(new(domain.RepositoriesService), new(*github.RepositoriesService)),
			wire.Bind(new(download.GitHubContentAPI), new(*github.RepositoriesService)),
		),
		wire.NewSet(
			registry.New,
			wire.Bind(new(domain.RegistryInstaller), new(*registry.Installer)),
		),
		wire.NewSet(
			download.NewGitHubContentFileDownloader,
			wire.Bind(new(domain.GitHubContentFileDownloader), new(*download.GitHubContentFileDownloader)),
		),
		wire.NewSet(
			reader.New,
			wire.Bind(new(domain.ConfigReader), new(*reader.ConfigReader)),
		),
		osenv.New,
		afero.NewOsFs,
		download.NewHTTPDownloader,
		wire.NewSet(
			link.New,
			wire.Bind(new(domain.Linker), new(*link.Linker)),
		),
	)
	return nil
}

func InitializeExecCommandController(ctx context.Context, param *config.Param, httpClient *http.Client, rt *runtime.Runtime) *cexec.Controller {
	wire.Build(
		cexec.New,
		wire.NewSet(
			finder.NewConfigFinder,
			wire.Bind(new(which.ConfigFinder), new(*finder.ConfigFinder)),
		),
		wire.NewSet(
			download.NewPackageDownloader,
			wire.Bind(new(domain.PackageDownloader), new(*download.PackageDownloader)),
		),
		wire.NewSet(
			installpackage.New,
			wire.Bind(new(domain.PackageInstaller), new(*installpackage.Installer)),
		),
		wire.NewSet(
			github.New,
			wire.Bind(new(domain.RepositoriesService), new(*github.RepositoriesService)),
			wire.Bind(new(download.GitHubContentAPI), new(*github.RepositoriesService)),
		),
		wire.NewSet(
			registry.New,
			wire.Bind(new(domain.RegistryInstaller), new(*registry.Installer)),
		),
		wire.NewSet(
			download.NewGitHubContentFileDownloader,
			wire.Bind(new(domain.GitHubContentFileDownloader), new(*download.GitHubContentFileDownloader)),
		),
		wire.NewSet(
			reader.New,
			wire.Bind(new(domain.ConfigReader), new(*reader.ConfigReader)),
		),
		wire.NewSet(
			which.New,
			wire.Bind(new(domain.WhichController), new(*which.Controller)),
		),
		wire.NewSet(
			exec.New,
			wire.Bind(new(installpackage.Executor), new(*exec.Executor)),
			wire.Bind(new(cexec.Executor), new(*exec.Executor)),
		),
		wire.NewSet(
			download.NewChecksumDownloader,
			wire.Bind(new(domain.ChecksumDownloader), new(*download.ChecksumDownloader)),
		),
		osenv.New,
		afero.NewOsFs,
		wire.NewSet(
			link.New,
			wire.Bind(new(domain.Linker), new(*link.Linker)),
		),
		download.NewHTTPDownloader,
		wire.NewSet(
			checksum.NewCalculator,
			wire.Bind(new(installpackage.ChecksumCalculator), new(*checksum.Calculator)),
		),
		wire.NewSet(
			unarchive.New,
			wire.Bind(new(installpackage.Unarchiver), new(*unarchive.Unarchiver)),
		),
		wire.NewSet(
			policy.NewChecker,
			wire.Bind(new(domain.PolicyChecker), new(*policy.Checker)),
		),
		wire.NewSet(
			policy.NewConfigReader,
			wire.Bind(new(domain.PolicyConfigReader), new(*policy.ConfigReader)),
		),
	)
	return &cexec.Controller{}
}

func InitializeUpdateAquaCommandController(ctx context.Context, param *config.Param, httpClient *http.Client, rt *runtime.Runtime) *updateaqua.Controller {
	wire.Build(
		updateaqua.New,
		afero.NewOsFs,
		wire.NewSet(
			github.New,
			wire.Bind(new(domain.RepositoriesService), new(*github.RepositoriesService)),
			wire.Bind(new(updateaqua.RepositoriesService), new(*github.RepositoriesService)),
		),
		wire.NewSet(
			installpackage.New,
			wire.Bind(new(updateaqua.AquaInstaller), new(*installpackage.Installer)),
		),
		download.NewHTTPDownloader,
		wire.NewSet(
			download.NewPackageDownloader,
			wire.Bind(new(domain.PackageDownloader), new(*download.PackageDownloader)),
		),
		wire.NewSet(
			exec.New,
			wire.Bind(new(installpackage.Executor), new(*exec.Executor)),
		),
		wire.NewSet(
			unarchive.New,
			wire.Bind(new(installpackage.Unarchiver), new(*unarchive.Unarchiver)),
		),
		wire.NewSet(
			checksum.NewCalculator,
			wire.Bind(new(installpackage.ChecksumCalculator), new(*checksum.Calculator)),
		),
		wire.NewSet(
			download.NewChecksumDownloader,
			wire.Bind(new(domain.ChecksumDownloader), new(*download.ChecksumDownloader)),
		),
		wire.NewSet(
			link.New,
			wire.Bind(new(domain.Linker), new(*link.Linker)),
		),
		wire.NewSet(
			policy.NewChecker,
			wire.Bind(new(domain.PolicyChecker), new(*policy.Checker)),
		),
	)
	return &updateaqua.Controller{}
}

func InitializeCopyCommandController(ctx context.Context, param *config.Param, httpClient *http.Client, rt *runtime.Runtime) *cp.Controller {
	wire.Build(
		cp.New,
		wire.NewSet(
			finder.NewConfigFinder,
			wire.Bind(new(which.ConfigFinder), new(*finder.ConfigFinder)),
			wire.Bind(new(install.ConfigFinder), new(*finder.ConfigFinder)),
		),
		wire.NewSet(
			install.New,
			wire.Bind(new(cp.Installer), new(*install.Controller)),
		),
		wire.NewSet(
			download.NewPackageDownloader,
			wire.Bind(new(domain.PackageDownloader), new(*download.PackageDownloader)),
		),
		wire.NewSet(
			installpackage.New,
			wire.Bind(new(domain.PackageInstaller), new(*installpackage.Installer)),
			wire.Bind(new(cp.PackageInstaller), new(*installpackage.Installer)),
		),
		wire.NewSet(
			github.New,
			wire.Bind(new(domain.RepositoriesService), new(*github.RepositoriesService)),
			wire.Bind(new(download.GitHubContentAPI), new(*github.RepositoriesService)),
		),
		wire.NewSet(
			registry.New,
			wire.Bind(new(domain.RegistryInstaller), new(*registry.Installer)),
		),
		wire.NewSet(
			download.NewGitHubContentFileDownloader,
			wire.Bind(new(domain.GitHubContentFileDownloader), new(*download.GitHubContentFileDownloader)),
		),
		wire.NewSet(
			reader.New,
			wire.Bind(new(domain.ConfigReader), new(*reader.ConfigReader)),
		),
		wire.NewSet(
			which.New,
			wire.Bind(new(domain.WhichController), new(*which.Controller)),
		),
		wire.NewSet(
			exec.New,
			wire.Bind(new(installpackage.Executor), new(*exec.Executor)),
			wire.Bind(new(cexec.Executor), new(*exec.Executor)),
		),
		wire.NewSet(
			download.NewChecksumDownloader,
			wire.Bind(new(domain.ChecksumDownloader), new(*download.ChecksumDownloader)),
		),
		osenv.New,
		afero.NewOsFs,
		wire.NewSet(
			link.New,
			wire.Bind(new(domain.Linker), new(*link.Linker)),
		),
		download.NewHTTPDownloader,
		wire.NewSet(
			checksum.NewCalculator,
			wire.Bind(new(installpackage.ChecksumCalculator), new(*checksum.Calculator)),
		),
		wire.NewSet(
			unarchive.New,
			wire.Bind(new(installpackage.Unarchiver), new(*unarchive.Unarchiver)),
		),
		wire.NewSet(
			policy.NewChecker,
			wire.Bind(new(domain.PolicyChecker), new(*policy.Checker)),
		),
		wire.NewSet(
			policy.NewConfigReader,
			wire.Bind(new(domain.PolicyConfigReader), new(*policy.ConfigReader)),
		),
	)
	return &cp.Controller{}
}

func InitializeUpdateChecksumCommandController(ctx context.Context, param *config.Param, httpClient *http.Client, rt *runtime.Runtime) *updatechecksum.Controller {
	wire.Build(
		updatechecksum.New,
		wire.NewSet(
			finder.NewConfigFinder,
			wire.Bind(new(updatechecksum.ConfigFinder), new(*finder.ConfigFinder)),
		),
		wire.NewSet(
			reader.New,
			wire.Bind(new(domain.ConfigReader), new(*reader.ConfigReader)),
		),
		wire.NewSet(
			download.NewChecksumDownloader,
			wire.Bind(new(domain.ChecksumDownloader), new(*download.ChecksumDownloader)),
		),
		wire.NewSet(
			registry.New,
			wire.Bind(new(domain.RegistryInstaller), new(*registry.Installer)),
		),
		wire.NewSet(
			github.New,
			wire.Bind(new(domain.RepositoriesService), new(*github.RepositoriesService)),
			wire.Bind(new(download.GitHubContentAPI), new(*github.RepositoriesService)),
		),
		wire.NewSet(
			download.NewGitHubContentFileDownloader,
			wire.Bind(new(domain.GitHubContentFileDownloader), new(*download.GitHubContentFileDownloader)),
		),
		wire.NewSet(
			download.NewPackageDownloader,
			wire.Bind(new(domain.PackageDownloader), new(*download.PackageDownloader)),
		),
		download.NewHTTPDownloader,
		afero.NewOsFs,
	)
	return &updatechecksum.Controller{}
}
