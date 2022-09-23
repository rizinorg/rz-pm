package pkg

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimplePackage(t *testing.T) {
	p := RizinPackage{
		Name:        "simple",
		Description: "simple description",
		Repo:        "https://github.com/rizinorg/jsdec",
	}

	tmpPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest")
	require.Nil(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)

	installPath, err := ioutil.TempDir(os.TempDir(), "rzpmtest-install")
	require.Nil(t, err, "temp path should be created")
	defer os.RemoveAll(tmpPath)

	err = p.Download(tmpPath)
	assert.Nil(t, err, "simple package should be downloaded")

	err = p.Install(installPath)
	assert.Nil(t, err, "simple package should be installed")
}
