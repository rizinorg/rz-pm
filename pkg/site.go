package pkg

import (
	"fmt"
	"os"
	"path/filepath"
)

const RZPM_DB_REPO_URL = "https://github.com/rizinorg/rz-pm-db"

type Site struct {
	Path     string
	Database Database
}

const dbDir string = "rz-pm-db"

func InitSite(path string) (Site, error) {
	// create the filesystem structure
	dbSubdir := filepath.Join(path, dbDir)
	paths := []string{
		path,
		dbSubdir,
	}

	for _, p := range paths {
		if err := os.MkdirAll(p, 0755); err != nil {
			return Site{}, fmt.Errorf("could not create %s: %w", p, err)
		}
	}

	d, err := InitDatabase(dbSubdir)
	if err != nil {
		return Site{}, err
	}

	s := Site{
		Path:     path,
		Database: d,
	}
	return s, nil
}

func (s Site) ListAvailablePackages() ([]RizinPackage, error) {
	return s.Database.ListAvailablePackages()
}

func (s Site) Remove() error {
	return os.RemoveAll(s.Path)
}
