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
	PackageName        string             `yaml:"name"`
	PackageVersion     string             `yaml:"version"`
	PackageDescription string             `yaml:"description"`
	PackageSource      RizinPackageSource `yaml:"source"`
}

type Package interface {
	Name() string
	Version() string
	Description() string
	Source() RizinPackageSource
	Download(baseArtifactsPath string) error
	Build(site Site) error
	Install(site Site) error
	Uninstall(site Site) error
}

func (rp RizinPackage) Name() string {
	return rp.PackageName
}

func (rp RizinPackage) Version() string {
	return rp.PackageVersion
}

func (rp RizinPackage) Description() string {
	return rp.PackageDescription
}

func (rp RizinPackage) Source() RizinPackageSource {
	return rp.PackageSource
}

// Download the source code of a package and extract it in the provided path
func (rp RizinPackage) Download(baseArtifactsPath string) error {
	artifactsPath := rp.artifactsPath(baseArtifactsPath)
	err := os.MkdirAll(artifactsPath, os.FileMode(0755))
	if err != nil {
		return err
	}

	fmt.Printf("Downloading %s source archive...\n", rp.PackageName)
	client := http.Client{}
	resp, err := client.Get(rp.PackageSource.URL)
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
	fmt.Printf("Verifying downloaded archive...\n")
	computedHash := sha256.Sum256(content)
	if hex.EncodeToString(computedHash[:]) != rp.PackageSource.Hash {
		return ErrRizinPackageWrongHash
	}

	tarballFileOpen.Seek(0, 0)
	var fileReader io.ReadCloser = tarballFileOpen
	if strings.HasSuffix(rp.PackageSource.URL, ".gz") {
		fileReader, err = gzip.NewReader(tarballFileOpen)
		if err != nil {
			return err
		}
		defer fileReader.Close()
	}

	fmt.Printf("Extracting %s code...\n", rp.PackageName)
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
	fmt.Printf("Source code for %s downloaded and extracted.\n", rp.PackageName)

	return nil
}

func (rp RizinPackage) artifactsPath(baseArtifactsPath string) string {
	return filepath.Join(baseArtifactsPath, rp.PackageName, rp.PackageVersion)
}
func (rp RizinPackage) sourcePath(baseArtifactsPath string) string {
	return filepath.Join(rp.artifactsPath(baseArtifactsPath), rp.PackageSource.Directory)
}

func (rp RizinPackage) buildMeson(site Site) error {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	args := rp.PackageSource.BuildArguments
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
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("meson", "compile", "-C", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (rp RizinPackage) installMeson(site Site) error {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	cmd := exec.Command("meson", "install", "-C", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (rp RizinPackage) uninstallMeson(site Site) error {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	cmd := exec.Command("ninja", "uninstall", "-C", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// Build a package if a source is provided
func (rp RizinPackage) Build(site Site) error {
	fmt.Printf("Building %s...\n", rp.PackageName)
	if rp.PackageSource.BuildSystem == "meson" {
		_, err := exec.LookPath("meson")
		if err != nil {
			return fmt.Errorf("make sure 'meson' is installed and in PATH")
		}

		return rp.buildMeson(site)
	} else {
		log.Printf("BuildSystem %s is not supported yet.", rp.PackageSource.BuildSystem)
		return fmt.Errorf("unsupported build system")
	}
}

// Install a package after building it
func (rp RizinPackage) Install(site Site) error {
	err := rp.Build(site)
	if err != nil {
		return err
	}

	fmt.Printf("Installing %s...\n", rp.PackageName)
	if rp.PackageSource.BuildSystem == "meson" {
		err = rp.installMeson(site)
	} else {
		log.Printf("BuildSystem %s is not supported yet.", rp.PackageSource.BuildSystem)
		err = fmt.Errorf("unsupported build system")
	}
	if err != nil {
		return err
	}
	fmt.Printf("Package %s built and installed.\n", rp.PackageName)
	return nil
}

func (rp RizinPackage) Uninstall(site Site) error {
	fmt.Printf("Uninstalling %s...\n", rp.PackageName)
	var err error
	if rp.PackageSource.BuildSystem == "meson" {
		err = rp.uninstallMeson(site)
	} else {
		log.Printf("BuildSystem %s is not supported yet.", rp.PackageSource.BuildSystem)
		err = fmt.Errorf("unsupported build system")
	}
	if err != nil {
		return err
	}
	fmt.Printf("Package %s uninstalled.\n", rp.PackageName)
	return nil
}
