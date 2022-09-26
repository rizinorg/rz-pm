package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const RZPM_DB_REPO_URL = "https://github.com/rizinorg/rz-pm-db"

type Site interface {
	ListAvailablePackages() ([]RizinPackage, error)
	GetPackage(name string) (RizinPackage, error)
	GetBaseDir() string
	GetArtifactsDir() string
	GetPkgConfigDir() string
	GetCMakeDir() string
	InstallPackage(pkg RizinPackage) error
	UninstallPackage(pkg RizinPackage) error
	DownloadPackage(pkg RizinPackage) error
	Remove() error
}

type RizinSite struct {
	Path          string
	Database      Database
	PkgConfigPath string
	CMakePath     string
}

const dbDir string = "rz-pm-db"
const artifactsDir string = "artifacts"

func InitSite(path string) (Site, error) {
	// create the filesystem structure
	dbSubdir := filepath.Join(path, dbDir)
	artifactsSubdir := filepath.Join(path, artifactsDir)
	paths := []string{
		path,
		dbSubdir,
		artifactsSubdir,
	}

	for _, p := range paths {
		if err := os.MkdirAll(p, 0755); err != nil {
			return RizinSite{}, fmt.Errorf("could not create %s: %w", p, err)
		}
	}

	d, err := InitDatabase(dbSubdir)
	if err != nil {
		return RizinSite{}, err
	}

	pkgConfigPath, err := getPkgConfigPath()
	if err != nil {
		return RizinSite{}, err
	}

	cmakePath, err := getCMakePath()
	if err != nil {
		return RizinSite{}, err
	}

	s := RizinSite{
		Path:          path,
		Database:      d,
		PkgConfigPath: pkgConfigPath,
		CMakePath:     cmakePath,
	}
	return s, nil
}

func (s RizinSite) ListAvailablePackages() ([]RizinPackage, error) {
	return s.Database.ListAvailablePackages()
}

func (s RizinSite) GetPackage(name string) (RizinPackage, error) {
	return s.Database.GetPackage(name)
}

func (s RizinSite) GetBaseDir() string {
	return s.Path
}

func (s RizinSite) GetArtifactsDir() string {
	return filepath.Join(s.Path, artifactsDir)
}

func (s RizinSite) GetPkgConfigDir() string {
	return s.PkgConfigPath
}

func (s RizinSite) GetCMakeDir() string {
	return s.CMakePath
}

func (s RizinSite) InstallPackage(pkg RizinPackage) error {
	return pkg.Install(s)
}

func (s RizinSite) UninstallPackage(pkg RizinPackage) error {
	return pkg.Uninstall(s)
}

func (s RizinSite) DownloadPackage(pkg RizinPackage) error {
	return pkg.Download(filepath.Join(s.Path, artifactsDir))
}

func (s RizinSite) Remove() error {
	return os.RemoveAll(s.Path)
}

func getRizinLibPath() (string, error) {
	cmd := exec.Command("rizin", "-H", "RZ_LIBDIR")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(string(out), "\n"), nil
}

func getPkgConfigPath() (string, error) {
	libPath, err := getRizinLibPath()
	if err != nil {
		return "", err
	}
	pkgConfigPath := filepath.Join(libPath, "pkgconfig")
	if _, err := os.Stat(pkgConfigPath); os.IsNotExist(err) {
		return "", err
	}
	return pkgConfigPath, nil
}

func getCMakePath() (string, error) {
	libPath, err := getRizinLibPath()
	if err != nil {
		return "", err
	}
	cmakePath := filepath.Join(libPath, "cmake")
	if _, err := os.Stat(cmakePath); os.IsNotExist(err) {
		return "", err
	}
	return cmakePath, nil
}
