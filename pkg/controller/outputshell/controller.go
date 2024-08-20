package outputshell

import (
	"context"
	"io"

	"github.com/aquaproj/aqua/v2/pkg/checksum"
	"github.com/aquaproj/aqua/v2/pkg/config"
	"github.com/aquaproj/aqua/v2/pkg/config/aqua"
	"github.com/aquaproj/aqua/v2/pkg/config/registry"
	"github.com/aquaproj/aqua/v2/pkg/runtime"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type Controller struct {
	rootDir           string
	configFinder      ConfigFinder
	configReader      ConfigReader
	registryInstaller RegistryInstaller
	fs                afero.Fs
	runtime           *runtime.Runtime
	stdout            io.Writer
}

type ConfigFinder interface {
	Finds(wd, configFilePath string) []string
}

type Shell struct {
	Env *Env `json:"env,omitempty"`
}

func (s *Shell) GetPaths() []string {
	if s == nil {
		return nil
	}
	return s.Env.GetPaths()
}

type Env struct {
	Path *Path `json:"PATH,omitempty"`
}

func (e *Env) GetPaths() []string {
	if e == nil {
		return nil
	}
	return e.Path.GetPaths()
}

type Path struct {
	Values []string `json:"values,omitempty"`
}

func (p *Path) GetPaths() []string {
	if p == nil {
		return nil
	}
	return p.Values
}

func New(param *config.Param, configFinder ConfigFinder, configReader ConfigReader, registInstaller RegistryInstaller, fs afero.Fs, rt *runtime.Runtime, stdout io.Writer) *Controller {
	return &Controller{
		rootDir:           param.RootDir,
		configFinder:      configFinder,
		configReader:      configReader,
		registryInstaller: registInstaller,
		fs:                fs,
		runtime:           rt,
		stdout:            stdout,
	}
}

type ConfigReader interface {
	Read(logE *logrus.Entry, configFilePath string, cfg *aqua.Config) error
}

type RegistryInstaller interface {
	InstallRegistries(ctx context.Context, logE *logrus.Entry, cfg *aqua.Config, cfgFilePath string, checksums *checksum.Checksums) (map[string]*registry.Config, error)
}