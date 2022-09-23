package pkg

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func containsPackage(packages []RizinPackage, name string) bool {
	for _, rp := range packages {
		if rp.Name == name {
			return true
		}
	}
	return false
}

func TestEmptySite(t *testing.T) {
	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.Nil(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)
	site, err := InitSite(tmpPath)
	require.Nil(t, err, "site should be initialized in tmpPath")
	assert.Equal(t, tmpPath, site.Path, "site path should be tmpPath")
	_, err = os.Stat(filepath.Join(tmpPath, "rz-pm-db"))
	assert.Nil(t, err, "rz-pm database directory should be there")
	_, err = os.Stat(filepath.Join(tmpPath, "rz-pm-db", "README.md"))
	assert.Nil(t, err, "rz-pm-db repository should be downloaded")
	_, err = os.Stat(filepath.Join(tmpPath, "rz-pm-db", "db"))
	assert.Nil(t, err, "rz-pm-db repository should be downloaded 2")
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
