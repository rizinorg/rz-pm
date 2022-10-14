package pkg

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadSimplePackage(t *testing.T) {
	p := RizinPackage{
		PackageName:        "simple",
		PackageDescription: "simple description",
		PackageVersion:     "0.0.1",
		PackageSource: RizinPackageSource{
			URL:            "https://github.com/rizinorg/jsdec/archive/refs/tags/v0.4.0.tar.gz",
			Hash:           "5afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d",
			BuildSystem:    "meson",
			Directory:      "p",
			BuildArguments: []string{"-Djsc_folder=.."},
		},
	}

	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.NoError(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)

	err = p.Download(tmpPath)
	assert.NoError(t, err, "simple package should be downloaded")
	_, err = os.Stat(filepath.Join(tmpPath, "simple", "0.0.1", "jsdec-0.4.0"))
	assert.NoError(t, err, "jsdec release should be downloaded and extracted")
	_, err = os.Stat(filepath.Join(tmpPath, "simple", "0.0.1", "jsdec-0.4.0", "p"))
	assert.NoError(t, err, "jsdec/p should be there")
}

func TestWrongHash(t *testing.T) {
	p := RizinPackage{
		PackageName:        "simple",
		PackageDescription: "simple description",
		PackageVersion:     "0.0.1",
		PackageSource: RizinPackageSource{
			URL:            "https://github.com/rizinorg/jsdec/archive/refs/tags/v0.4.0.tar.gz",
			Hash:           "sha256:6afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d",
			BuildSystem:    "meson",
			Directory:      "p",
			BuildArguments: []string{"-Djsc_folder=.."},
		},
	}

	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.Nil(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)

	installPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest-install")
	require.Nil(t, err, "install path should be created")
	defer os.RemoveAll(installPath)

	err = p.Download(tmpPath)
	assert.ErrorIs(t, err, ErrRizinPackageWrongHash, "wrong hash should be detected")
}

type FakeSite struct {
	ArtifactsDir string
}

func (s FakeSite) ListAvailablePackages() ([]Package, error) {
	return []Package{}, nil
}
func (s FakeSite) ListInstalledPackages() ([]Package, error) {
	return []Package{}, nil
}
func (s FakeSite) IsPackageInstalled(pkg Package) bool {
	return false
}
func (s FakeSite) GetPackage(name string) (Package, error) {
	return RizinPackage{}, nil
}
func (s FakeSite) GetBaseDir() string {
	return ""
}
func (s FakeSite) GetArtifactsDir() string {
	return s.ArtifactsDir
}
func (s FakeSite) GetPkgConfigDir() string {
	return "pkg-config-dir"
}
func (s FakeSite) GetCMakeDir() string {
	return ""
}
func (s FakeSite) InstallPackage(pkg Package) error {
	return nil
}
func (s FakeSite) UninstallPackage(pkg Package) error {
	return nil
}
func (s FakeSite) Remove() error {
	return nil
}

func TestInstallSimplePackage(t *testing.T) {
	log.SetOutput(os.Stderr)
	p := RizinPackage{
		PackageName:        "simple",
		PackageDescription: "simple description",
		PackageVersion:     "0.0.1",
		PackageSource: RizinPackageSource{
			URL:            "https://github.com/rizinorg/jsdec/archive/refs/tags/v0.4.0.tar.gz",
			Hash:           "5afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d",
			BuildSystem:    "meson",
			Directory:      "jsdec-0.4.0/p",
			BuildArguments: []string{"-Djsc_folder=..", "-Drizin_plugdir="},
		},
	}

	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.NoError(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)

	pluginsPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest-install")
	require.NoError(t, err, "install path should be created")
	defer os.RemoveAll(pluginsPath)
	p.PackageSource.BuildArguments[1] += pluginsPath

	err = p.Download(tmpPath)
	require.NoError(t, err, "package should be downloaded")

	err = p.Install(FakeSite{ArtifactsDir: tmpPath})
	assert.NoError(t, err, "The plugin should be built and installed without errors")
	files, err := ioutil.ReadDir(pluginsPath)
	require.NoError(t, err, "pluginsPath should be read")
	require.True(t, len(files) >= 1, "there should be one plugin installed")
	for i := range files {
		assert.Contains(t, files[i].Name(), "core_pdd", "the name of the plugin lib is jsdec")
	}
}

func TestUninstallSimplePackage(t *testing.T) {
	log.SetOutput(os.Stderr)
	p := RizinPackage{
		PackageName:        "simple",
		PackageDescription: "simple description",
		PackageVersion:     "0.0.1",
		PackageSource: RizinPackageSource{
			URL:            "https://github.com/rizinorg/jsdec/archive/refs/tags/v0.4.0.tar.gz",
			Hash:           "5afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d",
			BuildSystem:    "meson",
			Directory:      "jsdec-0.4.0/p",
			BuildArguments: []string{"-Djsc_folder=..", "-Drizin_plugdir="},
		},
	}

	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.NoError(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)

	pluginsPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest-install")
	require.NoError(t, err, "install path should be created")
	defer os.RemoveAll(pluginsPath)
	p.PackageSource.BuildArguments[1] += pluginsPath

	err = p.Download(tmpPath)
	require.NoError(t, err, "package should be downloaded")

	s := FakeSite{ArtifactsDir: tmpPath}
	err = p.Install(s)
	assert.NoError(t, err, "The plugin should be built and installed without errors")

	err = p.Uninstall(s)
	assert.NoError(t, err, "The plugin should be uninstalled without errors")

	files, err := ioutil.ReadDir(pluginsPath)
	require.NoError(t, err, "pluginsPath should be read")
	require.Len(t, files, 0, "there should be one plugins installed")
}
