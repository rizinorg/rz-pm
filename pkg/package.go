package pkg

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rizinorg/rz-pm/internal/util/dir"
)

type BuildSystem string

const (
	Meson BuildSystem = "meson"
)

type RizinPackageSource struct {
	URL            string
	Hash           string
	BuildSystem    BuildSystem `yaml:"build_system"`
	BuildArguments []string    `yaml:"build_arguments"`
	Directory      string
}

type RizinPackage struct {
	Name        string
	Version     string
	Description string
	Source      RizinPackageSource
}

// Download the source code of a package and extract it in the provided path
func (rp RizinPackage) Download(baseArtifactsPath string) error {
	artifactsPath := rp.artifactsPath(baseArtifactsPath)
	err := os.MkdirAll(artifactsPath, os.FileMode(0755))
	if err != nil {
		return err
	}

	client := http.Client{}
	resp, err := client.Get(rp.Source.URL)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	tarballFile, err := os.CreateTemp(artifactsPath, "")
	if err != nil {
		return err
	}
	defer os.Remove(tarballFile.Name())

	_, err = io.Copy(tarballFile, resp.Body)
	if err != nil {
		return err
	}

	tarballFileOpen, err := os.Open(tarballFile.Name())
	if err != nil {
		return err
	}

	content, err := io.ReadAll(tarballFileOpen)
	if err != nil {
		return err
	}
	computedHash := sha256.Sum256(content)
	if hex.EncodeToString(computedHash[:]) != rp.Source.Hash {
		return ErrRizinPackageWrongHash
	}

	tarballFileOpen.Seek(0, 0)
	var fileReader io.ReadCloser = tarballFileOpen
	if strings.HasSuffix(rp.Source.URL, ".gz") {
		fileReader, err = gzip.NewReader(tarballFileOpen)
		if err != nil {
			return err
		}
		defer fileReader.Close()
	}

	tarballReader := tar.NewReader(fileReader)
	for {
		header, err := tarballReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		filename := filepath.Join(artifactsPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(filename, os.FileMode(header.Mode))

			if err != nil {
				return err
			}
		case tar.TypeReg:
			// handle normal file
			writer, err := os.Create(filename)
			if err != nil {
				return err
			}

			io.Copy(writer, tarballReader)

			err = os.Chmod(filename, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			writer.Close()
		}
	}

	return nil
}

func (rp RizinPackage) artifactsPath(baseArtifactsPath string) string {
	return filepath.Join(baseArtifactsPath, rp.Name, rp.Version)
}
func (rp RizinPackage) sourcePath(baseArtifactsPath string) string {
	return filepath.Join(rp.artifactsPath(baseArtifactsPath), rp.Source.Directory)
}

func (rp RizinPackage) buildMeson(site Site) error {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	args := rp.Source.BuildArguments
	args = append(args, fmt.Sprintf("--prefix=%s/.local", dir.HomeDir()))
	if site.GetPkgConfigDir() != "" {
		args = append(args, fmt.Sprintf("--pkg-config-path=%s", site.GetPkgConfigDir()))
	}
	if site.GetCMakeDir() != "" {
		args = append(args, fmt.Sprintf("--cmake-prefix-path=%s", site.GetCMakeDir()))
	}
	args = append(args, "build")
	cmd := exec.Command("meson", args...)
	cmd.Dir = srcPath
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("meson", "compile", "-C", "build")
	cmd.Dir = srcPath
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (rp RizinPackage) installMeson(site Site) error {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	cmd := exec.Command("meson", "install", "-C", "build")
	cmd.Dir = srcPath
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (rp RizinPackage) uninstallMeson(site Site) error {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	cmd := exec.Command("ninja", "uninstall", "-C", "build")
	cmd.Dir = srcPath
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// Build a package if a source is provided
func (rp RizinPackage) Build(site Site) error {
	if rp.Source.BuildSystem == "meson" {
		return rp.buildMeson(site)
	} else {
		log.Printf("BuildSystem %s is not supported yet.", rp.Source.BuildSystem)
		return fmt.Errorf("unsupported build system")
	}
}

// Install a package after building it
func (rp RizinPackage) Install(site Site) error {
	if rp.Source.BuildSystem == "meson" {
		err := rp.Build(site)
		if err != nil {
			return err
		}
		return rp.installMeson(site)
	} else {
		log.Printf("BuildSystem %s is not supported yet.", rp.Source.BuildSystem)
		return fmt.Errorf("unsupported build system")
	}
}

func (rp RizinPackage) Uninstall(site Site) error {
	if rp.Source.BuildSystem == "meson" {
		return rp.uninstallMeson(site)
	} else {
		log.Printf("BuildSystem %s is not supported yet.", rp.Source.BuildSystem)
		return fmt.Errorf("unsupported build system")
	}
}
