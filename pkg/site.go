package pkg

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
)

const (
	SiteDirEnvVar = "RZPM_SITEDIR"
)

func SiteDir() string {
	if envVar := os.Getenv(SiteDirEnvVar); envVar != "" {
		return envVar
	}

	return filepath.Join(xdg.DataHome, "rz-pm", "site")
}

const RZPM_DB_REPO_URL = "https://github.com/rizinorg/rz-pm-db"

type Site interface {
	ListAvailablePackages() ([]Package, error)
	ListInstalledPackages() ([]Package, error)
	IsPackageInstalled(pkg Package) bool
	GetPackage(name string) (Package, error)
	GetBaseDir() string
	GetArtifactsDir() string
	GetPkgConfigDir() string
	GetCMakeDir() string
	InstallPackage(pkg Package) error
	UninstallPackage(pkg Package) error
	Remove() error
}

type RizinSite struct {
	Path                   string
	Database               Database
	PkgConfigPath          string
	CMakePath              string
	installedPackagesNames []string
}

const dbDir string = "rz-pm-db"
const artifactsDir string = "artifacts"
const installedFile string = "installed"

func InitSite(path string) (Site, error) {
	// create the filesystem structure
	dbSubdir := filepath.Join(path, dbDir)
	artifactsSubdir := filepath.Join(path, artifactsDir)
	installedFilePath := filepath.Join(path, installedFile)
	paths := []string{
		path,
		dbSubdir,
		artifactsSubdir,
	}

	for _, p := range paths {
		if err := os.MkdirAll(p, 0755); err != nil {
			return &RizinSite{}, fmt.Errorf("could not create %s: %w", p, err)
		}
	}

	installedPackageNames, err := getInstalledPackageNames(installedFilePath)
	if err != nil {
		return &RizinSite{}, err
	}

	d, err := InitDatabase(dbSubdir)
	if err != nil {
		return &RizinSite{}, err
	}

	pkgConfigPath, err := getPkgConfigPath()
	if err != nil {
		return &RizinSite{}, err
	}

	cmakePath, err := getCMakePath()
	if err != nil {
		return &RizinSite{}, err
	}

	s := RizinSite{
		Path:                   path,
		Database:               d,
		PkgConfigPath:          pkgConfigPath,
		CMakePath:              cmakePath,
		installedPackagesNames: installedPackageNames,
	}
	return &s, nil
}

func (s *RizinSite) ListAvailablePackages() ([]Package, error) {
	return s.Database.ListAvailablePackages()
}

func (s *RizinSite) ListInstalledPackages() ([]Package, error) {
	installedPackages := make([]Package, len(s.installedPackagesNames))
	for i := range s.installedPackagesNames {
		pkg, err := s.Database.GetPackage(s.installedPackagesNames[i])
		if err != nil {
			return nil, err
		}
		installedPackages[i] = pkg
	}
	return installedPackages, nil
}

func (s *RizinSite) IsPackageInstalled(pkg Package) bool {
	return containsString(s.installedPackagesNames, pkg.Name())
}

func (s *RizinSite) GetPackage(name string) (Package, error) {
	return s.Database.GetPackage(name)
}

func (s *RizinSite) GetBaseDir() string {
	return s.Path
}

func (s *RizinSite) GetArtifactsDir() string {
	return filepath.Join(s.Path, artifactsDir)
}

func (s *RizinSite) GetPkgConfigDir() string {
	return s.PkgConfigPath
}

func (s *RizinSite) GetCMakeDir() string {
	return s.CMakePath
}

func (s *RizinSite) InstallPackage(pkg Package) error {
	if containsString(s.installedPackagesNames, pkg.Name()) {
		return fmt.Errorf("package %s already installed", pkg.Name())
	}

	err := pkg.Install(s)
	if err != nil {
		return err
	}

	s.installedPackagesNames = append(s.installedPackagesNames, pkg.Name())
	installedFilePath := filepath.Join(s.Path, installedFile)
	return updateInstalledPackageNames(installedFilePath, s.installedPackagesNames)
}

func (s *RizinSite) UninstallPackage(pkg Package) error {
	if !containsString(s.installedPackagesNames, pkg.Name()) {
		return fmt.Errorf("package %s not installed", pkg.Name())
	}

	err := pkg.Uninstall(s)
	if err != nil {
		return err
	}

	s.installedPackagesNames = removeStringFromSlice(s.installedPackagesNames, pkg.Name())
	installedFilePath := filepath.Join(s.Path, installedFile)
	return updateInstalledPackageNames(installedFilePath, s.installedPackagesNames)
}

func (s *RizinSite) Remove() error {
	return os.RemoveAll(s.Path)
}

func getRizinLibPath() (string, error) {
	if _, err := exec.LookPath("rizin"); err != nil {
		return "", fmt.Errorf("rizin does not seem to be installed on your system. Make sure it is installed and in PATH")
	}
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
	_, err = os.Stat(pkgConfigPath)
	if os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
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
	_, err = os.Stat(cmakePath)
	if os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return cmakePath, nil
}

func getInstalledPackageNames(path string) ([]string, error) {
	by, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return []string{}, nil
	} else if err != nil {
		return []string{}, err
	}

	var v []string
	err = json.Unmarshal(by, &v)
	if err != nil {
		return []string{}, err
	}

	return v, nil
}

func updateInstalledPackageNames(path string, names []string) error {
	by, err := json.Marshal(names)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, by, fs.FileMode(0622))
	if err != nil {
		return err
	}
	return err
}

func removeStringFromSlice(sl []string, name string) []string {
	for i := range sl {
		if sl[i] == name {
			return append(sl[:i], sl[i+1:]...)
		}
	}
	return sl
}

func containsString(sl []string, name string) bool {
	for _, v := range sl {
		if v == name {
			return true
		}
	}
	return false
}
