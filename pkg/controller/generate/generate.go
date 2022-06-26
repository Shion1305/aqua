package generate

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/clivm/clivm/pkg/config"
	finder "github.com/clivm/clivm/pkg/config-finder"
	reader "github.com/clivm/clivm/pkg/config-reader"
	"github.com/clivm/clivm/pkg/config/aqua"
	"github.com/clivm/clivm/pkg/config/registry"
	"github.com/clivm/clivm/pkg/expr"
	githubSvc "github.com/clivm/clivm/pkg/github"
	instregst "github.com/clivm/clivm/pkg/install-registry"
	"github.com/google/go-github/v44/github"
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/logrus-error/logerr"
	"gopkg.in/yaml.v2"
)

type Controller struct {
	stdin                   io.Reader
	stdout                  io.Writer
	gitHubRepositoryService githubSvc.RepositoryService
	registryInstaller       instregst.Installer
	configFinder            finder.ConfigFinder
	configReader            reader.ConfigReader
	fuzzyFinder             FuzzyFinder
	fs                      afero.Fs
}

func New(configFinder finder.ConfigFinder, configReader reader.ConfigReader, registInstaller instregst.Installer, gh githubSvc.RepositoryService, fs afero.Fs, fuzzyFinder FuzzyFinder) *Controller {
	return &Controller{
		stdin:                   os.Stdin,
		stdout:                  os.Stdout,
		configFinder:            configFinder,
		configReader:            configReader,
		registryInstaller:       registInstaller,
		gitHubRepositoryService: gh,
		fs:                      fs,
		fuzzyFinder:             fuzzyFinder,
	}
}

// Generate searches packages in registries and outputs the configuration to standard output.
// If no package is specified, the interactive fuzzy finder is launched.
// If the package supports, the latest version is gotten by GitHub API.
func (ctrl *Controller) Generate(ctx context.Context, logE *logrus.Entry, param *config.Param, args ...string) error {
	cfgFilePath, err := ctrl.configFinder.Find(param.PWD, param.ConfigFilePath, param.GlobalConfigFilePaths...)
	if err != nil {
		return err //nolint:wrapcheck
	}

	list, err := ctrl.generate(ctx, logE, param, cfgFilePath, args...)
	if err != nil {
		return err
	}
	if list == nil {
		return nil
	}
	if !param.Insert {
		if err := yaml.NewEncoder(ctrl.stdout).Encode(list); err != nil {
			return fmt.Errorf("output generated package configuration: %w", err)
		}
		return nil
	}

	return ctrl.generateInsert(cfgFilePath, list)
}

type FindingPackage struct {
	PackageInfo  *registry.PackageInfo
	RegistryName string
}

func (ctrl *Controller) generate(ctx context.Context, logE *logrus.Entry, param *config.Param, cfgFilePath string, args ...string) (interface{}, error) {
	cfg := &aqua.Config{}
	if err := ctrl.configReader.Read(cfgFilePath, cfg); err != nil {
		return nil, err //nolint:wrapcheck
	}
	registryContents, err := ctrl.registryInstaller.InstallRegistries(ctx, cfg, cfgFilePath, logE)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	if param.File != "" || len(args) != 0 {
		return ctrl.outputListedPkgs(ctx, logE, param, registryContents, args...)
	}

	// maps the package and the registry
	var pkgs []*FindingPackage
	for registryName, registryContent := range registryContents {
		for _, pkg := range registryContent.PackageInfos.ToMapWarn(logE) {
			pkgs = append(pkgs, &FindingPackage{
				PackageInfo:  pkg,
				RegistryName: registryName,
			})
		}
	}

	// Launch the fuzzy finder
	idxes, err := ctrl.fuzzyFinder.Find(pkgs)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return nil, nil //nolint:nilnil
		}
		return nil, fmt.Errorf("find the package: %w", err)
	}
	arr := make([]interface{}, len(idxes))
	for i, idx := range idxes {
		arr[i] = ctrl.getOutputtedPkg(ctx, pkgs[idx], logE)
	}

	return arr, nil
}

func getGeneratePkg(s string) string {
	if !strings.Contains(s, ",") {
		return "standard," + s
	}
	return s
}

func (ctrl *Controller) outputListedPkgs(ctx context.Context, logE *logrus.Entry, param *config.Param, registryContents map[string]*registry.Config, pkgNames ...string) (interface{}, error) {
	m := map[string]*FindingPackage{}
	for registryName, registryContent := range registryContents {
		logE := logE.WithField("registry_name", registryName)
		for pkgName, pkg := range registryContent.PackageInfos.ToMapWarn(logE) {
			logE := logE.WithField("package_name", pkgName)
			m[registryName+","+pkgName] = &FindingPackage{
				PackageInfo:  pkg,
				RegistryName: registryName,
			}
			for _, alias := range pkg.Aliases {
				if alias.Name == "" {
					logE.Warn("ignore a package alias because the alias is empty")
					continue
				}
				m[registryName+","+alias.Name] = &FindingPackage{
					PackageInfo:  pkg,
					RegistryName: registryName,
				}
			}
		}
	}

	outputPkgs := []*aqua.Package{}
	for _, pkgName := range pkgNames {
		pkgName = getGeneratePkg(pkgName)
		findingPkg, ok := m[pkgName]
		if !ok {
			return nil, logerr.WithFields(errUnknownPkg, logrus.Fields{"package_name": pkgName}) //nolint:wrapcheck
		}
		outputPkg := ctrl.getOutputtedPkg(ctx, findingPkg, logE)
		outputPkgs = append(outputPkgs, outputPkg)
	}

	if param.File != "" {
		pkgs, err := ctrl.readGeneratedPkgsFromFile(ctx, param, outputPkgs, m, logE)
		if err != nil {
			return nil, err
		}
		outputPkgs = pkgs
	}
	return outputPkgs, nil
}

func (ctrl *Controller) readGeneratedPkgsFromFile(ctx context.Context, param *config.Param, outputPkgs []*aqua.Package, m map[string]*FindingPackage, logE *logrus.Entry) ([]*aqua.Package, error) {
	var file io.Reader
	if param.File == "-" {
		file = ctrl.stdin
	} else {
		f, err := ctrl.fs.Open(param.File)
		if err != nil {
			return nil, fmt.Errorf("open the package list file: %w", err)
		}
		defer f.Close()
		file = f
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := getGeneratePkg(scanner.Text())
		findingPkg, ok := m[txt]
		if !ok {
			return nil, logerr.WithFields(errUnknownPkg, logrus.Fields{"package_name": txt}) //nolint:wrapcheck
		}
		outputPkg := ctrl.getOutputtedPkg(ctx, findingPkg, logE)
		outputPkgs = append(outputPkgs, outputPkg)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the file: %w", err)
	}
	return outputPkgs, nil
}

func (ctrl *Controller) listAndGetTagName(ctx context.Context, pkgInfo *registry.PackageInfo, logE *logrus.Entry) string {
	repoOwner := pkgInfo.RepoOwner
	repoName := pkgInfo.RepoName
	opt := &github.ListOptions{
		PerPage: 30, //nolint:gomnd
	}
	versionFilter, err := expr.CompileVersionFilter(*pkgInfo.VersionFilter)
	if err != nil {
		return ""
	}
	for {
		releases, _, err := ctrl.gitHubRepositoryService.ListReleases(ctx, repoOwner, repoName, opt)
		if err != nil {
			logerr.WithError(logE, err).WithFields(logrus.Fields{
				"repo_owner": repoOwner,
				"repo_name":  repoName,
			}).Warn("list releases")
			return ""
		}
		for _, release := range releases {
			if release.GetPrerelease() {
				continue
			}
			f, err := expr.EvaluateVersionFilter(versionFilter, release.GetTagName())
			if err != nil || !f {
				continue
			}
			return release.GetTagName()
		}
		if len(releases) != opt.PerPage {
			return ""
		}
		opt.Page++
	}
}

func (ctrl *Controller) listAndGetTagNameFromTag(ctx context.Context, pkgInfo *registry.PackageInfo, logE *logrus.Entry) string {
	repoOwner := pkgInfo.RepoOwner
	repoName := pkgInfo.RepoName
	opt := &github.ListOptions{
		PerPage: 30, //nolint:gomnd
	}
	versionFilter, err := expr.CompileVersionFilter(*pkgInfo.VersionFilter)
	if err != nil {
		return ""
	}
	for {
		tags, _, err := ctrl.gitHubRepositoryService.ListTags(ctx, repoOwner, repoName, opt)
		if err != nil {
			logerr.WithError(logE, err).WithFields(logrus.Fields{
				"repo_owner": repoOwner,
				"repo_name":  repoName,
			}).Warn("list releases")
			return ""
		}
		for _, tag := range tags {
			tagName := tag.GetName()
			f, err := expr.EvaluateVersionFilter(versionFilter, tagName)
			if err != nil || !f {
				continue
			}
			return tagName
		}
		if len(tags) != opt.PerPage {
			return ""
		}
		opt.Page++
	}
}

func (ctrl *Controller) getOutputtedGitHubPkgFromTag(ctx context.Context, outputPkg *aqua.Package, pkgInfo *registry.PackageInfo, logE *logrus.Entry) {
	repoOwner := pkgInfo.RepoOwner
	repoName := pkgInfo.RepoName
	var tagName string
	if pkgInfo.VersionFilter != nil {
		tagName = ctrl.listAndGetTagNameFromTag(ctx, pkgInfo, logE)
	} else {
		tags, _, err := ctrl.gitHubRepositoryService.ListTags(ctx, repoOwner, repoName, nil)
		if err != nil {
			logerr.WithError(logE, err).WithFields(logrus.Fields{
				"repo_owner": repoOwner,
				"repo_name":  repoName,
			}).Warn("list GitHub tags")
			return
		}
		if len(tags) == 0 {
			return
		}
		tag := tags[0]
		tagName = tag.GetName()
	}

	if pkgName := pkgInfo.GetName(); pkgName == repoOwner+"/"+repoName || strings.HasPrefix(pkgName, repoOwner+"/"+repoName+"/") {
		outputPkg.Name += "@" + tagName
		outputPkg.Version = ""
	} else {
		outputPkg.Version = tagName
	}
}

func (ctrl *Controller) getOutputtedGitHubPkg(ctx context.Context, outputPkg *aqua.Package, pkgInfo *registry.PackageInfo, logE *logrus.Entry) {
	if pkgInfo.VersionSource == "github_tag" {
		ctrl.getOutputtedGitHubPkgFromTag(ctx, outputPkg, pkgInfo, logE)
		return
	}
	repoOwner := pkgInfo.RepoOwner
	repoName := pkgInfo.RepoName
	var tagName string
	if pkgInfo.VersionFilter != nil {
		tagName = ctrl.listAndGetTagName(ctx, pkgInfo, logE)
	} else {
		release, _, err := ctrl.gitHubRepositoryService.GetLatestRelease(ctx, repoOwner, repoName)
		if err != nil {
			logerr.WithError(logE, err).WithFields(logrus.Fields{
				"repo_owner": repoOwner,
				"repo_name":  repoName,
			}).Warn("get the latest release")
			return
		}
		tagName = release.GetTagName()
	}
	if pkgName := pkgInfo.GetName(); pkgName == repoOwner+"/"+repoName || strings.HasPrefix(pkgName, repoOwner+"/"+repoName+"/") {
		outputPkg.Name += "@" + tagName
		outputPkg.Version = ""
	} else {
		outputPkg.Version = tagName
	}
}

func (ctrl *Controller) getOutputtedPkg(ctx context.Context, pkg *FindingPackage, logE *logrus.Entry) *aqua.Package {
	outputPkg := &aqua.Package{
		Name:     pkg.PackageInfo.GetName(),
		Registry: pkg.RegistryName,
		Version:  "[SET PACKAGE VERSION]",
	}
	if outputPkg.Registry == "standard" {
		outputPkg.Registry = ""
	}
	if ctrl.gitHubRepositoryService == nil {
		return outputPkg
	}
	if pkgInfo := pkg.PackageInfo; pkgInfo.HasRepo() {
		ctrl.getOutputtedGitHubPkg(ctx, outputPkg, pkgInfo, logE)
		return outputPkg
	}
	return outputPkg
}
