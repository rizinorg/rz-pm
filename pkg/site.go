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
	GetPackageFromFile(filename string) (Package, error)
	GetInstalledPackage(name string) (InstalledPackage, error)
	GetBaseDir() string
	GetArtifactsDir() string
	GetPkgConfigDir() string
	GetCMakeDir() string
	InstallPackage(pkg Package) error
	UninstallPackage(pkg Package) error
	CleanPackage(pkg Package) error
	Remove() error
	RizinVersion() string
}

type InstalledPackage struct {
	InstalledName  string    `json:"name"`
	InstalledFiles *[]string `json:"files"`
	RizinVersion   *string   `json:"rizin_version"`
}

type RizinSite struct {
	Path              string
	Database          Database
	PkgConfigPath     string
	CMakePath         string
	installedPackages []InstalledPackage
	rizinVersion      string
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

	rizinVersion, err := getRizinVersion()
	if err != nil {
		return &RizinSite{}, err
	}
	installedPackages, err := getInstalledPackages(installedFilePath, rizinVersion)
	if err != nil {
		return &RizinSite{}, err
	}

	d, err := InitDatabase(dbSubdir, rizinVersion)
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
		Path:              path,
		Database:          d,
		PkgConfigPath:     pkgConfigPath,
		CMakePath:         cmakePath,
		installedPackages: installedPackages,
		rizinVersion:      rizinVersion,
	}
	return &s, nil
}

func (rp InstalledPackage) Name() string {
	return rp.InstalledName
}
func (rp InstalledPackage) Version() string            { return "" }
func (rp InstalledPackage) Description() string        { return "" }
func (rp InstalledPackage) Summary() string            { return "" }
func (rp InstalledPackage) Source() RizinPackageSource { return RizinPackageSource{} }
func (rp InstalledPackage) Download(baseArtifactsPath string) error {
	return fmt.Errorf("cannot be called")
}
func (rp InstalledPackage) Build(site Site) error { return fmt.Errorf("cannot be called") }
func (rp InstalledPackage) Install(site Site) ([]string, error) {
	return nil, fmt.Errorf("cannot be called")
}
func (rp InstalledPackage) Uninstall(site Site) error { return fmt.Errorf("cannot be called") }

func (s *RizinSite) ListAvailablePackages() ([]Package, error) {
	res, err := s.Database.ListAvailablePackages()
	if err != nil {
		return []Package{}, err
	}

	for i := range s.installedPackages {
		_, err := s.Database.GetPackage(s.installedPackages[i].InstalledName)
		if err != nil {
			res = append(res, s.installedPackages[i])
		}
	}

	return res, nil
}

func (s *RizinSite) ListInstalledPackages() ([]Package, error) {
	installedPackages := make([]Package, len(s.installedPackages))
	for i := range s.installedPackages {
		pkg, err := s.Database.GetPackage(s.installedPackages[i].InstalledName)
		if err != nil {
			installedPackages[i] = s.installedPackages[i]
		} else {
			installedPackages[i] = pkg
		}
	}
	return installedPackages, nil
}

func (s *RizinSite) RizinVersion() string {
	return s.rizinVersion
}

func (s *RizinSite) IsPackageInstalled(pkg Package) bool {
	return s.ContainsInstalledPackage(pkg.Name())
}

func (s *RizinSite) GetPackage(name string) (Package, error) {
	return s.Database.GetPackage(name)
}

func (s *RizinSite) GetPackageFromFile(filename string) (Package, error) {
	return ParsePackageFile(filename)
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
	if s.ContainsInstalledPackage(pkg.Name()) {
		return fmt.Errorf("package %s already installed", pkg.Name())
	}

	files, err := pkg.Install(s)
	if err != nil {
		return err
	}

	minorVersion := GetMajorMinorVersion(s.rizinVersion)
	s.installedPackages = append(s.installedPackages, InstalledPackage{
		pkg.Name(),
		&files,
		&minorVersion,
	})
	installedFilePath := filepath.Join(s.Path, installedFile)
	return updateInstalledPackages(installedFilePath, s.installedPackages)
}

func (s *RizinSite) UninstallPackage(pkg Package) error {
	if !s.ContainsInstalledPackage(pkg.Name()) {
		return fmt.Errorf("package %s not installed", pkg.Name())
	}

	installedPackage, err := s.GetInstalledPackage(pkg.Name())
	if err != nil {
		return err
	}

	if installedPackage.InstalledFiles == nil {
		// NOTE: kept for compatibility with v0.1.9
		err = pkg.Uninstall(s)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("Uninstalling %s...\n", pkg.Name())
		for _, file := range *installedPackage.InstalledFiles {
			os.RemoveAll(file)
		}

	}

	s.installedPackages = removePackageFromSlice(s.installedPackages, pkg.Name())
	fmt.Printf("Package %s uninstalled.\n", pkg.Name())

	installedFilePath := filepath.Join(s.Path, installedFile)
	return updateInstalledPackages(installedFilePath, s.installedPackages)
}

func (s *RizinSite) CleanPackage(pkg Package) error {
	pkgArtifactsPath := filepath.Join(s.GetArtifactsDir(), pkg.Name(), pkg.Version())
	_, err := os.Stat(pkgArtifactsPath)
	if err != nil {
		return fmt.Errorf("package %s does not have any build artifacts", pkg.Name())
	}

	err = os.RemoveAll(pkgArtifactsPath)
	if err != nil {
		return fmt.Errorf("failed to remove build artifacts for package %s", pkg.Name())
	}

	return nil
}

func (s *RizinSite) Remove() error {
	return os.RemoveAll(s.Path)
}

func getRizinVersion() (string, error) {
	if _, err := exec.LookPath("rizin"); err != nil {
		return "", fmt.Errorf("rizin does not seem to be installed on your system. Make sure it is installed and in PATH")
	}
	cmd := exec.Command("rizin", "-H", "RZ_VERSION")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(out), "\r\n"), nil
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

	return strings.TrimRight(string(out), "\r\n"), nil
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

func getInstalledPackages(path string, rizinVersion string) ([]InstalledPackage, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return []InstalledPackage{}, nil
	}

	by, err := ioutil.ReadFile(path)
	if err != nil {
		return []InstalledPackage{}, err
	}

	var v []InstalledPackage
	err = json.Unmarshal(by, &v)
	if err != nil {
		var vs []string
		err = json.Unmarshal(by, &vs)
		if err != nil {
			return []InstalledPackage{}, err
		}

		v = []InstalledPackage{}
		for _, s := range vs {
			if s != "" {
				v = append(v, InstalledPackage{InstalledName: s, InstalledFiles: nil, RizinVersion: nil})
			}
		}
	}

	version := GetMajorMinorVersion(rizinVersion)
	for i := range v {
		if v[i].RizinVersion == nil {
			v[i].RizinVersion = &version
		}
	}

	return v, nil
}

func updateInstalledPackages(path string, packages []InstalledPackage) error {
	by, err := json.Marshal(packages)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, by, fs.FileMode(0622))
	if err != nil {
		return err
	}
	return err
}

func removePackageFromSlice(sl []InstalledPackage, name string) []InstalledPackage {
	for i := range sl {
		if sl[i].InstalledName == name {
			ret := make([]InstalledPackage, 0)
			if i > 0 {
				ret = append(ret, sl[:i]...)
			}
			if i < len(sl)-1 {
				ret = append(ret, sl[i+1:]...)
			}
			return ret
		}
	}
	return sl
}

func (s *RizinSite) GetInstalledPackage(name string) (InstalledPackage, error) {
	for _, v := range s.installedPackages {
		if v.InstalledName == name {
			return v, nil
		}
	}
	return InstalledPackage{}, fmt.Errorf("installed package %s not found", name)
}

func (s *RizinSite) ContainsInstalledPackage(name string) bool {
	_, err := s.GetInstalledPackage(name)
	return err == nil
}
