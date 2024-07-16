package installpackage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aquaproj/aqua/v2/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/logrus-error/logerr"
)

func (is *Installer) createLinks(logE *logrus.Entry, pkgs []*config.Package) bool {
	failed := false
	for _, pkg := range pkgs {
		pkgInfo := pkg.PackageInfo
		for _, file := range pkgInfo.GetFiles() {
			if isWindows(is.runtime.GOOS) {
				if err := is.createHardLink(filepath.Join(is.rootDir, "bin", file.Name+".exe"), filepath.Join(is.rootDir, proxyName+".exe"), logE); err != nil {
					logerr.WithError(logE, err).Error("create the proxy file")
					failed = true
				}
				continue
			}
			if err := is.createLink(filepath.Join(is.rootDir, "bin", file.Name), filepath.Join("..", proxyName), logE); err != nil {
				logerr.WithError(logE, err).Error("create the symbolic link")
				failed = true
				continue
			}
		}
	}
	return failed
}

func (is *Installer) replaceWithHardlinks() error {
	hardlinkFile := filepath.Join(is.rootDir, "hardlink")
	if f, err := afero.Exists(is.fs, hardlinkFile); err != nil {
		return fmt.Errorf("check if a hardlink flag exists: %w", err)
	} else if f {
		return nil
	}
	binDir := filepath.Join(is.rootDir, "bin")
	infos, err := afero.ReadDir(is.fs, binDir)
	if err != nil {
		return fmt.Errorf("read a bin dir: %w", err)
	}
	proxy := filepath.Join(is.rootDir, "aqua-proxy")
	for _, info := range infos {
		p := filepath.Join(binDir, info.Name())
		if err := is.fs.Remove(p); err != nil {
			return fmt.Errorf("remove a file to replace it with a hard link: %w", err)
		}
		if err := is.linker.Hardlink(proxy, p); err != nil {
			return fmt.Errorf("create a hard link: %w", err)
		}
	}

	if f, err := is.fs.Create(hardlinkFile); err != nil {
		return fmt.Errorf("create a hardlink flag: %w", err)
	} else { //nolint:revive
		f.Close()
	}
	return nil
}

func (is *Installer) createHardLink(linkPath, linkDest string, logE *logrus.Entry) error {
	logE.WithFields(logrus.Fields{
		"command": filepath.Base(linkPath),
	}).Info("creating a hard link")
	if err := is.linker.Hardlink(linkDest, linkPath); err != nil {
		return fmt.Errorf("create a hard link: %w", err)
	}
	return nil
}

func (is *Installer) createLink(linkPath, linkDest string, logE *logrus.Entry) error {
	if fileInfo, err := is.linker.Lstat(linkPath); err == nil {
		switch mode := fileInfo.Mode(); {
		case mode.IsDir():
			// if file is a directory, raise error
			return fmt.Errorf("%s has already existed and is a directory", linkPath)
		case mode&os.ModeNamedPipe != 0:
			// if file is a pipe, raise error
			return fmt.Errorf("%s has already existed and is a named pipe", linkPath)
		case mode.IsRegular():
			// if file is a regular file, remove it and create a symlink.
			if err := is.fs.Remove(linkPath); err != nil {
				return fmt.Errorf("remove a file to create a symbolic link (%s): %w", linkPath, err)
			}
			if err := is.linker.Symlink(linkDest, linkPath); err != nil {
				return fmt.Errorf("create a symbolic link: %w", err)
			}
			return nil
		case mode&os.ModeSymlink != 0:
			return is.recreateLink(linkPath, linkDest, logE)
		default:
			return fmt.Errorf("unexpected file mode %s: %s", linkPath, mode.String())
		}
	}
	logE.WithFields(logrus.Fields{
		"command": filepath.Base(linkPath),
	}).Info("create a symbolic link")
	if err := is.linker.Symlink(linkDest, linkPath); err != nil {
		return fmt.Errorf("create a symbolic link: %w", err)
	}
	return nil
}

func (is *Installer) recreateLink(linkPath, linkDest string, logE *logrus.Entry) error {
	lnDest, err := is.linker.Readlink(linkPath)
	if err != nil {
		return fmt.Errorf("read a symbolic link (%s): %w", linkPath, err)
	}
	if linkDest == lnDest {
		return nil
	}
	// recreate link
	logE.WithFields(logrus.Fields{
		// TODO add version
		"link_file": linkPath,
		"old":       lnDest,
		"new":       linkDest,
	}).Debug("recreate a symbolic link")
	if err := is.fs.Remove(linkPath); err != nil {
		return fmt.Errorf("remove a symbolic link (%s): %w", linkPath, err)
	}
	if err := is.linker.Symlink(linkDest, linkPath); err != nil {
		return fmt.Errorf("create a symbolic link: %w", err)
	}
	return nil
}
