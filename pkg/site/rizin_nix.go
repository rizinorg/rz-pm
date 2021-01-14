// +build darwin freebsd linux openbsd

package site

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/rizinorg/rz-pm/pkg/git"
	"github.com/rizinorg/rz-pm/pkg/process"
)

func (s Site) InstallRizin(prefix, version string) error {
	srcDir := filepath.Join(s.gitSubDir(), "rizin")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf(
			"could not create the filesystem tree for %s: %v",
			srcDir,
			err)
	}

	log.Print("Opening " + srcDir)

	repo, err := git.Open(srcDir)
	if err != nil {
		log.Printf("Could not open %s as a git repo: %v", srcDir, err)
		log.Print("Running git init")

		if repo, err = git.Init(srcDir, false); err != nil {
			return fmt.Errorf("could not run git init: %v", err)
		}

		origin := "https://github.com/rizinorg/rizin"

		log.Print("Setting the origin to " + origin)
		if err = repo.AddRemote("origin", origin); err != nil {
			return fmt.Errorf("could not set origin: %v", err)
		}
	}

	if err := repo.Fetch(); err != nil {
		return fmt.Errorf("Could not fetch: %v", err)
	}

	if err := repo.Checkout(version); err != nil {
		return fmt.Errorf("Could not checkout %q: %v", version, err)
	}

	if err := repo.UpdateSubmodules(); err != nil {
		return fmt.Errorf("Could not update submodules: %v", err)
	}

	if err := repo.Pull("origin", version, []string{"--recurse-submodules"}); err != nil {
		return err
	}

	buildDir := fmt.Sprintf("build-%s", version)
	prefixArg := fmt.Sprintf("--prefix=%s", prefix)
	if res, err := process.Run("meson", []string{prefixArg, "--buildtype=release", buildDir}, srcDir); err != nil {
		log.Print(string(res.Stdout.String()))
		return err
	}

	if res, err := process.Run("ninja", []string{"-C", buildDir}, srcDir); err != nil {
		log.Print(string(res.Stdout.String()))
		return err
	}

	if _, err := process.Run("ninja", []string{"-C", buildDir, "install"}, srcDir); err != nil {
		return err
	}

	return nil
}
