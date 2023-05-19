package pkg

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"
)

type Database struct {
	Path string
}

var ErrRizinPackageWrongHash = errors.New("wrong hash")

const dbPath string = "db"

func InitDatabase(path string) (Database, error) {
	d := Database{path}

	err := d.updateDatabase()
	if err != nil {
		return Database{}, fmt.Errorf("could not download the rz-pm database")
	}

	return d, nil
}

func (d Database) updateDatabase() error {
	repo, err := git.PlainOpen(d.Path)
	if err == git.ErrRepositoryNotExists {
		log.Printf("Downloading rz-pm-db repository...\n")
		repo, err = git.PlainClone(d.Path, false, &git.CloneOptions{
			URL: RZPM_DB_REPO_URL,
		})
	}
	if err != nil {
		return err
	}

	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	log.Printf("Updating rz-pm-db repository...\n")
	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != git.NoErrAlreadyUpToDate {
		return err
	}
	return nil
}

func ParsePackageFile(path string) (Package, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return RizinPackage{}, err
	}

	var p RizinPackage
	err = yaml.Unmarshal(content, &p)
	if err != nil {
		return RizinPackage{}, err
	}

	if p.PackageName == "" || p.PackageVersion == "" || p.PackageSummary == "" {
		return RizinPackage{}, fmt.Errorf("wrong file plugin format: name, version, and summary are mandatory")
	}
	if p.PackageSource != nil {
		if p.PackageSource.URL == "" || p.PackageSource.BuildSystem == "" {
			return RizinPackage{}, fmt.Errorf("wrong file plugin format: Source URL and Build System are mandatory")
		}
		if !p.isGitRepo() && p.PackageSource.Hash == "" {
			return RizinPackage{}, fmt.Errorf("wrong file plugin format: Source Hash is mandatory for non-git plugins")
		} else if p.isGitRepo() && p.PackageSource.Hash != "" {
			return RizinPackage{}, fmt.Errorf("wrong file plugin format: Source Hash should not be used for git plugins")
		}
	}
	return p, nil
}

func (d Database) ListAvailablePackages() ([]Package, error) {
	dbPath := filepath.Join(d.Path, dbPath)
	files, err := ioutil.ReadDir(dbPath)
	if err != nil {
		return nil, err
	}

	packages := []Package{}
	for _, file := range files {
		// skip directories
		if file.IsDir() {
			continue
		}

		name := filepath.Join(dbPath, file.Name())

		p, err := ParsePackageFile(name)
		if err != nil {
			fmt.Printf("Warning: could not read %s: %v\n", name, err)
			continue
		}

		packages = append(packages, p)
	}

	return packages, nil
}

func (d Database) GetPackage(name string) (Package, error) {
	packages, err := d.ListAvailablePackages()
	if err != nil {
		return RizinPackage{}, err
	}

	for _, pkg := range packages {
		if pkg.Name() == name {
			return pkg, nil
		}
	}

	return RizinPackage{}, fmt.Errorf("package '%s' not found", name)
}
