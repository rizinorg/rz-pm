package pkg

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func containsPackage(packages []Package, name string) bool {
	for _, rp := range packages {
		if rp.Name() == name {
			return true
		}
	}
	return false
}

func TestEmptySite(t *testing.T) {
	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.NoError(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)
	site, err := InitSite(tmpPath)
	require.NoError(t, err, "site should be initialized in tmpPath")
	assert.Equal(t, tmpPath, site.GetBaseDir(), "site path should be tmpPath")
	_, err = os.Stat(filepath.Join(tmpPath, "rz-pm-db"))
	assert.NoError(t, err, "rz-pm database directory should be there")
	_, err = os.Stat(filepath.Join(tmpPath, "rz-pm-db", "README.md"))
	assert.NoError(t, err, "rz-pm-db repository should be downloaded")
	_, err = os.Stat(filepath.Join(tmpPath, "rz-pm-db", "db"))
	assert.NoError(t, err, "rz-pm-db repository should be downloaded 2")
}

func TestExistingSite(t *testing.T) {
	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.Nil(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)
	_, err = InitSite(tmpPath)
	require.Nil(t, err, "site should be initialized when dir is empty")
	_, err = InitSite(tmpPath)
	assert.Nil(t, err, "site should be initialized even when dir is already initialized")
	_, err = os.Stat(filepath.Join(tmpPath, "rz-pm-db", "README.md"))
	assert.Nil(t, err, "rz-pm-db repository should be downloaded")
	_, err = os.Stat(filepath.Join(tmpPath, "rz-pm-db", "db"))
	assert.Nil(t, err, "rz-pm-db repository should be downloaded 2")
}

func TestListPackages(t *testing.T) {
	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.Nil(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)
	site, err := InitSite(tmpPath)
	require.Nil(t, err, "site should be initialized when dir is empty")

	packages, err := site.ListAvailablePackages()
	assert.Nil(t, err, "no errors while retrieving packages")
	assert.True(t, len(packages) > 0, "there should be at least one package in the database")
	assert.True(t, containsPackage(packages, "jsdec"), "jsdec package should be present in the database")
}

func TestGoodPackageFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "package-format")
	require.NoError(t, err, "temporary file should be created")
	defer tmpFile.Close()

	tmpFile.WriteString(`name: simple
version: 0.0.1
summary: simple description
source:
  url: https://github.com/rizinorg/jsdec
  hash: 5afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d
  build_system: meson
  build_arguments:
    - -Djsc_folder=..
  directory: jsdec-0.4.0/p
`)

	pkg, err := parsePackageFromFile(tmpFile.Name())
	require.NoError(t, err, "no errors in parsing the above package file")
	assert.Equal(t, "simple", pkg.Name())
	assert.Equal(t, "0.0.1", pkg.Version())
	assert.Equal(t, "simple description", pkg.Summary())
	assert.Equal(t, "https://github.com/rizinorg/jsdec", pkg.Source().URL)
	assert.Equal(t, "5afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d", pkg.Source().Hash)
	assert.Equal(t, Meson, pkg.Source().BuildSystem)
	assert.Contains(t, pkg.Source().BuildArguments, "-Djsc_folder=..")
	assert.Equal(t, "jsdec-0.4.0/p", pkg.Source().Directory)
}

func TestWrongPackageFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "package-format")
	require.NoError(t, err, "temporary file should be created")
	defer tmpFile.Close()

	f1 := `version: 0.0.1
summary: simple description
source:
  url: https://github.com/rizinorg/jsdec/archive/refs/tags/v0.4.0.tar.gz
  hash: 5afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d
  build_system: meson
  build_arguments:
    - -Djsc_folder=..
  directory: jsdec-0.4.0/p
`

	f2 := `name: simple
summary: simple description
source:
  url: https://github.com/rizinorg/jsdec/archive/refs/tags/v0.4.0.tar.gz
  hash: 5afe9a823c1c31ccf641dc1667a092418cd84f5cb9865730580783ca7c44e93d
  build_system: meson
  build_arguments:
    - -Djsc_folder=..
  directory: jsdec-0.4.0/p
`

	f3 := `name: simple
version: 0.0.1
summary: simple description
`

	tmpFile.WriteString(f1)

	_, err = parsePackageFromFile(tmpFile.Name())
	assert.Error(t, err, "missing name should fail parsing")

	tmpFile.Truncate(0)
	tmpFile.WriteString(f2)

	_, err = parsePackageFromFile(tmpFile.Name())
	assert.Error(t, err, "missing version should fail parsing")

	tmpFile.Truncate(0)
	tmpFile.WriteString(f3)

	_, err = parsePackageFromFile(tmpFile.Name())
	assert.Error(t, err, "missing source should fail parsing")
}

type FakePackage struct {
	myName string
}

func (fp FakePackage) Name() string {
	return fp.myName
}
func (fp FakePackage) Version() string {
	return ""
}
func (fp FakePackage) Summary() string {
	return ""
}
func (fp FakePackage) Description() string {
	return ""
}
func (fp FakePackage) Source() RizinPackageSource {
	return RizinPackageSource{}
}
func (fp FakePackage) Download(baseArtifactsPath string) error {
	return nil
}
func (fp FakePackage) Build(site Site) error {
	return nil
}
func (fp FakePackage) Install(site Site) error {
	return nil
}
func (fp FakePackage) Uninstall(site Site) error {
	return nil
}

func TestListInstalledPackages(t *testing.T) {
	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.Nil(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)
	site, err := InitSite(tmpPath)
	require.Nil(t, err, "site should be initialized when dir is empty")

	pkg := FakePackage{myName: "jsdec"}

	err = site.InstallPackage(pkg)
	require.NoError(t, err)

	packages, err := site.ListAvailablePackages()
	assert.NoError(t, err, "no errors while retrieving packages")
	assert.True(t, len(packages) > 0, "there should be at least one package in the database")

	installedPackages, err := site.ListInstalledPackages()
	assert.NoError(t, err, "no errors while retrieving installed packages")
	assert.Len(t, installedPackages, 1, "there should be just one package installed")
	assert.Equal(t, "jsdec", installedPackages[0].Name(), "jsdec package should be installed")
}
