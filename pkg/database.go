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
var ErrRizinPackageWrongFormat = errors.New("wrong format")

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

func parsePackageFromFile(path string) (RizinPackage, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return RizinPackage{}, err
	}

	var p RizinPackage
	err = yaml.Unmarshal(content, &p)
	if err != nil {
		return RizinPackage{}, err
	}

	if p.Name == "" || p.Version == "" || p.Source.URL == "" || p.Source.Hash == "" || p.Source.BuildSystem == "" {
		return RizinPackage{}, ErrRizinPackageWrongFormat
	}
	return p, nil
}

func (d Database) ListAvailablePackages() ([]RizinPackage, error) {
	dbPath := filepath.Join(d.Path, dbPath)
	files, err := ioutil.ReadDir(dbPath)
	if err != nil {
		return []RizinPackage{}, err
	}

	packages := []RizinPackage{}
	for _, file := range files {
		// skip directories
		if file.IsDir() {
			continue
		}

		name := filepath.Join(dbPath, file.Name())

		p, err := parsePackageFromFile(name)
		if err != nil {
			fmt.Printf("Warning: could not read %s: %v", name, err)
			continue
		}

		packages = append(packages, p)
	}

	return packages, nil
}

func (d Database) GetPackage(name string) (RizinPackage, error) {
	packages, err := d.ListAvailablePackages()
	if err != nil {
		return RizinPackage{}, err
	}

	for _, pkg := range packages {
		if pkg.Name == name {
			return pkg, nil
		}
	}

	return RizinPackage{}, fmt.Errorf("package not found")
}
