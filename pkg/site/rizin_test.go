// +build integration

package site

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRizin(t *testing.T) {
	siteDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(siteDir)

	s, err := New(siteDir)
	if err != nil {
		t.Fatal(err)
	}

	prefix, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(prefix)

	t.Run("InstallRizin", func(t *testing.T) {
		version := os.Getenv("RZ_VERSION")

		if version == "" {
			t.Fatal("The rizin version must be defined")
		}

		if err := s.InstallRizin(prefix, version); err != nil {
			t.Fatal(err)
		}

		rzBin := "rizin"

		if runtime.GOOS == "windows" {
			rzBin = "rizin.bat"
		}

		rzPath := filepath.Join(prefix, "bin", rzBin)

		if _, err := os.Stat(rzPath); err != nil {
			t.Fatalf("Could not stat(%q)", rzPath)
		}
	})

	t.Run("UninstallRizin", func(t *testing.T) {
		if err := s.UninstallRizin(prefix); err != nil {
			t.Fatal(err)
		}
	})
}
