package pkg

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"gopkg.in/yaml.v2"
)

type Database struct {
	Path string
}

var ErrRizinPackageWrongHash = errors.New("wrong hash")

const dbPath string = "db"

func InitDatabase(path string, rizinVersion string) (Database, error) {
	firstTime := false

	// if path does not exist, create it and force a db update
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0o755)
		if err != nil {
			return Database{}, fmt.Errorf("failed to create database directory: %w", err)
		}
		firstTime = true
	}

	d := Database{path}

	if firstTime {
		log.Printf("Initializing rz-pm-db repository at %s...\n", d.Path)
		err := d.UpdateDatabase(rizinVersion)
		if err != nil {
			return Database{}, fmt.Errorf("failed to initialize rz-pm-db repository: %w", err)
		}
	}

	return d, nil
}

func getBranchName(s string) plumbing.ReferenceName {
	return plumbing.ReferenceName("refs/remotes/origin/v" + s)
}

func remoteBranches(s storer.ReferenceStorer) (storer.ReferenceIter, error) {
	refs, err := s.IterReferences()
	if err != nil {
		return nil, err
	}

	return storer.NewReferenceFilteredIter(func(ref *plumbing.Reference) bool {
		return ref.Name().IsRemote()
	}, refs), nil
}

func (d Database) switchTag(repo *git.Repository, w *git.Worktree, rizinVersion string) (string, error) {
	branches, err := remoteBranches(repo.Storer)
	if err != nil {
		return "", err
	}

	versionPieces := strings.SplitN(rizinVersion, ".", 3)

	var switchBranch string
	var switchHash plumbing.Hash
	var switchBranchPrio int = math.MaxInt
	_ = branches.ForEach(func(b *plumbing.Reference) error {
		pieces := strings.Split(b.Name().String(), "/")
		branchName := pieces[len(pieces)-1]
		if branchName == "v"+rizinVersion && switchBranchPrio > 0 {
			switchBranch = branchName
			switchBranchPrio = 0
			switchHash = b.Hash()
		} else if branchName == "v"+versionPieces[0]+"."+versionPieces[1] && switchBranchPrio > 1 {
			switchBranch = branchName
			switchBranchPrio = 1
			switchHash = b.Hash()
		} else if branchName == "v"+versionPieces[0] && switchBranchPrio > 2 {
			switchBranch = branchName
			switchBranchPrio = 2
			switchHash = b.Hash()
		}
		return nil
	})

	if switchBranchPrio == math.MaxInt {
		return "", fmt.Errorf("could not find a tag for version %s", rizinVersion)
	}

	localBranchName := plumbing.ReferenceName("refs/heads/" + switchBranch)
	create := false
	if _, err := repo.Storer.Reference(localBranchName); err != nil {
		ref := plumbing.NewHashReference(localBranchName, switchHash)
		err = repo.Storer.SetReference(ref)
		if err != nil {
			return "", err
		}
		create = true
	}

	err = w.Checkout(&git.CheckoutOptions{Branch: localBranchName, Create: create})
	if err != nil {
		return "", err
	}
	return switchBranch, nil
}

func (d Database) UpdateDatabase(rizinVersion string) error {
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

	h, err := repo.Head()
	if err != nil {
		return err
	}

	branchName := h.Name().String()
	branchNamePieces := strings.Split(branchName, "/")
	branchName = strings.TrimLeft(branchNamePieces[len(branchNamePieces)-1], "v")

	if !strings.HasPrefix(rizinVersion, branchName) {
		tagName, err := d.switchTag(repo, w, rizinVersion)
		if err != nil {
			log.Printf("Failed to switch rz-pm-db to version %s, default to main branch", rizinVersion)
			err = w.Checkout(&git.CheckoutOptions{Branch: "refs/heads/master"})
			if err != nil {
				return err
			}
		} else {
			log.Printf("Switched rz-pm-db to %s...\n", tagName)
		}
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
	files, err := os.ReadDir(dbPath)
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
