package pkg

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/adrg/xdg"
	"github.com/go-git/go-git/v5"
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
	PackageName        string              `yaml:"name"`
	PackageVersion     string              `yaml:"version"`
	PackageSummary     string              `yaml:"summary"`
	PackageDescription string              `yaml:"description"`
	PackageSource      *RizinPackageSource `yaml:"source"`
}

type Package interface {
	Name() string
	Version() string
	Summary() string
	Description() string
	Source() RizinPackageSource
	Download(baseArtifactsPath string) error
	Build(site Site) error
	Install(site Site) ([]string, error)
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

func (rp RizinPackage) Summary() string {
	return rp.PackageSummary
}

func (rp RizinPackage) Source() RizinPackageSource {
	return *rp.PackageSource
}

func (rp RizinPackage) isGitRepo() bool {
	return rp.PackageSource == nil || strings.HasSuffix(rp.PackageSource.URL, ".git")
}

func (rp RizinPackage) isSupportedArchiveRepo() bool {
	return rp.PackageSource == nil || strings.HasSuffix(rp.PackageSource.URL, ".tar.gz") || strings.HasSuffix(rp.PackageSource.URL, ".tar")
}

func (rp RizinPackage) downloadTar(artifactsPath string) error {
	fmt.Printf("Downloading %s source archive...\n", rp.PackageName)
	client := http.Client{}
	resp, err := client.Get(rp.PackageSource.URL)
	if err != nil {
		return fmt.Errorf("failed to download package source: %w", err)
	}

	defer resp.Body.Close()
	tarballFile, err := os.CreateTemp(artifactsPath, "")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for package source: %w", err)
	}
	defer os.Remove(tarballFile.Name())

	_, err = io.Copy(tarballFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write package source to temporary file: %w", err)
	}

	tarballFileOpen, err := os.Open(tarballFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open temporary file for package source: %w", err)
	}

	content, err := io.ReadAll(tarballFileOpen)
	if err != nil {
		return fmt.Errorf("failed to read package source from temporary file: %w", err)
	}
	fmt.Printf("Verifying downloaded archive...\n")
	computedHash := sha256.Sum256(content)
	if hex.EncodeToString(computedHash[:]) != rp.PackageSource.Hash {
		fmt.Printf("Hash for downloaded archive does not match.\n")
		fmt.Printf("Expected: %s\n", rp.PackageSource.Hash)
		fmt.Printf("Actual: %s\n", hex.EncodeToString(computedHash[:]))
		return ErrRizinPackageWrongHash
	}

	tarballFileOpen.Seek(0, 0)
	var fileReader io.ReadCloser = tarballFileOpen
	if strings.HasSuffix(rp.PackageSource.URL, ".gz") {
		fileReader, err = gzip.NewReader(tarballFileOpen)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader for package source: %w", err)
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
			return fmt.Errorf("failed to read tarball header: %w", err)
		}

		filename := filepath.Join(artifactsPath, header.Name)
		cleanedFilename := filepath.Clean(filename)
		if !strings.HasPrefix(cleanedFilename, artifactsPath) {
			return fmt.Errorf("trying to extract a file outside the base path")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(filename, os.FileMode(header.Mode))

			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filename, err)
			}
		case tar.TypeReg:
			// handle normal file
			writer, err := os.Create(filename)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", filename, err)
			}

			io.Copy(writer, tarballReader)

			err = os.Chmod(filename, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to set permissions for file %s: %w", filename, err)
			}

			writer.Close()
		}
	}
	fmt.Printf("Source code for %s downloaded and extracted.\n", rp.PackageName)

	return nil
}

func (rp RizinPackage) downloadGit(artifactsPath string) error {
	gitProjectNamePieces := strings.Split(rp.PackageSource.URL, "/")
	gitProjectName := gitProjectNamePieces[len(gitProjectNamePieces)-1]
	gitProjectName = strings.TrimSuffix(gitProjectName, ".git")

	projectPath := filepath.Join(artifactsPath, gitProjectName)
	if fi, err := os.Stat(projectPath); !os.IsNotExist(err) && fi.IsDir() {
		repo, err := git.PlainOpen(projectPath)
		if err != nil {
			return fmt.Errorf("failed to open git repository at %s: %w", projectPath, err)
		}

		tree, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree for git repository at %s: %w", projectPath, err)
		}

		err = tree.Pull(&git.PullOptions{Progress: nil, RecurseSubmodules: git.DefaultSubmoduleRecursionDepth})
		if err == nil || err == git.NoErrAlreadyUpToDate {
			return nil
		}
		return fmt.Errorf("failed to pull git repository at %s: %w", projectPath, err)
	} else {
		_, err = git.PlainClone(projectPath, false, &git.CloneOptions{
			URL:               rp.PackageSource.URL,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		})
		return fmt.Errorf("failed to clone git repository from %s: %w", rp.PackageSource.URL, err)
	}
}

// Download the source code of a package and extract it in the provided path
func (rp RizinPackage) Download(baseArtifactsPath string) error {
	log.Printf("Downloading package %s... from '%s'", rp.PackageName, rp.PackageSource.URL)
	artifactsPath := rp.artifactsPath(baseArtifactsPath)
	err := os.MkdirAll(artifactsPath, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to create artifacts directory %s: %w", artifactsPath, err)
	}

	if rp.isSupportedArchiveRepo() {
		return rp.downloadTar(artifactsPath)
	} else if rp.isGitRepo() {
		return rp.downloadGit(artifactsPath)
	} else {
		return fmt.Errorf("source URL not supported! Use a .tar.gz/.tar/.git URL")
	}
}

func (rp RizinPackage) artifactsPath(baseArtifactsPath string) string {
	return filepath.Join(baseArtifactsPath, rp.PackageName, rp.PackageVersion)
}
func (rp RizinPackage) sourcePath(baseArtifactsPath string) string {
	return filepath.Join(rp.artifactsPath(baseArtifactsPath), rp.PackageSource.Directory)
}

func (rp RizinPackage) buildMeson(site Site) error {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	args := []string{"setup"}
	args = append(args, rp.PackageSource.BuildArguments...)
	args = append(args, fmt.Sprintf("--prefix=%s/.local", xdg.Home))
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

	log.Printf("Running meson setup:")
	log.Printf("\tdir: %s", srcPath)
	log.Printf("\targs: %s", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("ninja", "-C", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	log.Printf("Running ninja build...")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (rp RizinPackage) buildCMake(site Site) error {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	args := []string{}
	args = append(args, rp.PackageSource.BuildArguments...)
	args = append(args, fmt.Sprintf("-DCMAKE_INSTALL_PREFIX=%s/.local", xdg.Home))
	if site.GetCMakeDir() != "" {
		args = append(args, fmt.Sprintf("-DCMAKE_PREFIX_PATH=%s", site.GetCMakeDir()))
	}
	args = append(args, "-B")
	args = append(args, "build")
	cmd := exec.Command("cmake", args...)
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	log.Printf("Running cmake:")
	log.Printf("\tdir: %s", srcPath)
	log.Printf("\targs: %s", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure package %s: %w", rp.PackageName, err)
	}

	cmd = exec.Command("cmake", "--build", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	log.Printf("Running cmake build...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build package %s: %w", rp.PackageName, err)
	}

	return nil
}

func (rp RizinPackage) installMeson(site Site) ([]string, error) {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	cmd := exec.Command("meson", "install", "-C", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	cmd = exec.Command("meson", "introspect", "--installed", "build")
	cmd.Dir = srcPath
	cmd.Stderr = log.Writer()
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to introspect meson build: %w", err)
	}

	var data map[string]string
	err = json.Unmarshal([]byte(out), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse meson introspect output: %w", err)
	}

	var installedFiles []string
	for _, v := range data {
		installedFiles = append(installedFiles, v)
	}

	return installedFiles, nil
}

func (rp RizinPackage) installCMake(site Site) ([]string, error) {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	cmd := exec.Command("cmake", "--install", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	log.Printf("Running cmake install...")
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Join(srcPath, "build", "install_manifest.txt"))
	if err != nil {
		return nil, fmt.Errorf("could not open install_manifest.txt: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var installedFiles []string
	for scanner.Scan() {
		installedFiles = append(installedFiles, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading install_manifest.txt: %w", err)
	}

	return installedFiles, nil
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

func buildErrorMsg(msg string) string {
	if runtime.GOOS == "windows" {
		return "To build Rizin packages on Windows you need to enable the 'Developer Command Prompt for Visual Studio'. Follow the instructions at https://learn.microsoft.com/en-us/visualstudio/ide/reference/command-prompt-powershell?view=vs-2022 to install and execute it. Moreover " + msg
	}
	return msg
}

// Build a package if a source is provided
func (rp RizinPackage) Build(site Site) error {
	if site.GetPkgConfigDir() == "" && site.GetCMakeDir() == "" {
		return fmt.Errorf("make sure rizin development files are installed (e.g. librizin-dev, rizin-devel, etc.)")
	}

	srcPath := rp.sourcePath(site.GetArtifactsDir())
	fi, err := os.Stat(srcPath)
	if rp.isGitRepo() || err != nil || !fi.IsDir() {
		err := rp.Download(site.GetArtifactsDir())
		if err != nil {
			return err
		}
	}

	fmt.Printf("Building %s...\n", rp.PackageName)
	if rp.PackageSource.BuildSystem == "meson" {
		_, err := exec.LookPath("meson")
		if err != nil {
			return fmt.Errorf(buildErrorMsg("make sure 'meson' is installed and in PATH"))
		}

		_, err = exec.LookPath("pkg-config")
		if err != nil {
			_, err = exec.LookPath("cmake")
			if err != nil {
				return fmt.Errorf(buildErrorMsg("make sure either 'cmake' or `pkg-config` are installed and in PATH"))
			}
		}

		return rp.buildMeson(site)
	} else if rp.PackageSource.BuildSystem == "cmake" {
		_, err := exec.LookPath("cmake")
		if err != nil {
			return fmt.Errorf(buildErrorMsg("make sure 'cmake' is installed and in PATH"))
		}

		_, err = exec.LookPath("pkg-config")
		if err != nil {
			return fmt.Errorf(buildErrorMsg("make sure `pkg-config` is installed and in PATH"))
		}

		return rp.buildCMake(site)
	} else {
		log.Printf("BuildSystem %s is not supported yet.", rp.PackageSource.BuildSystem)
		return fmt.Errorf("unsupported build system")
	}
}

// Install a package after building it
func (rp RizinPackage) Install(site Site) ([]string, error) {
	err := rp.Build(site)
	if err != nil {
		return []string{}, err
	}

	var installed_files []string
	fmt.Printf("Installing %s...\n", rp.PackageName)
	if rp.PackageSource.BuildSystem == "meson" {
		installed_files, err = rp.installMeson(site)
	} else if rp.PackageSource.BuildSystem == "cmake" {
		installed_files, err = rp.installCMake(site)
	} else {
		log.Printf("BuildSystem %s is not supported yet.", rp.PackageSource.BuildSystem)
		err = fmt.Errorf("unsupported build system")
	}
	if err != nil {
		return []string{}, err
	}
	fmt.Printf("Package %s built and installed.\n", rp.PackageName)
	return installed_files, nil
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
