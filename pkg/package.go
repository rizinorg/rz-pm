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
	"time"

	"github.com/adrg/xdg"
	"github.com/go-git/go-git/v5"
)

type BuildSystem string

const (
	Meson BuildSystem = "meson"
	CMake BuildSystem = "cmake"
)

const gitProgressDotInterval = 2 * time.Second

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
	if rp.PackageSource == nil {
		return RizinPackageSource{}
	}
	return *rp.PackageSource
}

func (rp RizinPackage) isGitRepo() bool {
	return rp.PackageSource != nil && strings.HasSuffix(rp.PackageSource.URL, ".git")
}

func (rp RizinPackage) isSupportedArchiveRepo() bool {
	return rp.PackageSource != nil && (strings.HasSuffix(rp.PackageSource.URL, ".tar.gz") || strings.HasSuffix(rp.PackageSource.URL, ".tar"))
}

func (rp RizinPackage) validateSource() error {
	//fail here instead of letting nil source data break later steps
	if rp.PackageSource == nil {
		return fmt.Errorf("package %s does not define a source", rp.PackageName)
	}
	return nil
}

func secureJoin(basePath, relativePath string) (string, error) {
	cleanBase := filepath.Clean(basePath)
	targetPath := filepath.Clean(filepath.Join(cleanBase, relativePath))

	relPath, err := filepath.Rel(cleanBase, targetPath)
	if err != nil {
		return "", err
	}
	//reject paths that escape the extraction root, including sibling-prefix traversal cases.
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) || filepath.IsAbs(relPath) {
		return "", fmt.Errorf("trying to extract a file outside the base path")
	}
	return targetPath, nil
}

func gitProjectNameFromURL(url string) string {
	trimmedURL := strings.TrimSuffix(url, ".git")
	lastSeparator := strings.LastIndexAny(trimmedURL, `/\`)
	if lastSeparator == -1 {
		return trimmedURL
	}
	return trimmedURL[lastSeparator+1:]
}

func runWithDotProgress(message string, interval time.Duration, fn func() error) error {
	fmt.Print(message)

	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Print(".")
			}
		}
	}()

	err := fn()
	close(done)
	<-stopped
	fmt.Println()
	return err
}

func runCommandWithDotProgress(message string, cmd *exec.Cmd) error {
	return runWithDotProgress(message, gitProgressDotInterval, cmd.Run)
}

func (rp RizinPackage) downloadTar(artifactsPath string) error {
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

	err = runWithDotProgress(
		fmt.Sprintf("Downloading %s source archive...", rp.PackageName),
		gitProgressDotInterval,
		func() error {
			_, err := io.Copy(tarballFile, resp.Body)
			return err
		},
	)
	if err != nil {
		return err
	}

	tarballFileOpen, err := os.Open(tarballFile.Name())
	if err != nil {
		return err
	}
	defer tarballFileOpen.Close()

	err = runWithDotProgress(
		"Verifying downloaded archive...",
		gitProgressDotInterval,
		func() error {
			content, err := io.ReadAll(tarballFileOpen)
			if err != nil {
				return err
			}

			computedHash := sha256.Sum256(content)
			if hex.EncodeToString(computedHash[:]) != rp.PackageSource.Hash {
				fmt.Printf("Hash for downloaded archive does not match.\n")
				fmt.Printf("Expected: %s\n", rp.PackageSource.Hash)
				fmt.Printf("Actual: %s\n", hex.EncodeToString(computedHash[:]))
				return ErrRizinPackageWrongHash
			}
			return nil
		},
	)
	if err != nil {
		return err
	}

	_, err = tarballFileOpen.Seek(0, 0)
	if err != nil {
		return err
	}
	var fileReader io.ReadCloser = tarballFileOpen
	if strings.HasSuffix(rp.PackageSource.URL, ".gz") {
		fileReader, err = gzip.NewReader(tarballFileOpen)
		if err != nil {
			return err
		}
		defer fileReader.Close()
	}

	tarballReader := tar.NewReader(fileReader)
	err = runWithDotProgress(
		fmt.Sprintf("Extracting %s code...", rp.PackageName),
		gitProgressDotInterval,
		func() error {
			for {
				header, err := tarballReader.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				filename, err := secureJoin(artifactsPath, header.Name)
				if err != nil {
					return err
				}

				switch header.Typeflag {
				case tar.TypeDir:
					err = os.MkdirAll(filename, os.FileMode(header.Mode))
					if err != nil {
						return err
					}
				case tar.TypeReg:
					//create parent directories explicitly so nested archive entries extract reliably.
					err = os.MkdirAll(filepath.Dir(filename), 0755)
					if err != nil {
						return err
					}

					writer, err := os.Create(filename)
					if err != nil {
						return err
					}

					_, err = io.Copy(writer, tarballReader)
					closeErr := writer.Close()
					if err != nil {
						return err
					}
					if closeErr != nil {
						return closeErr
					}

					err = os.Chmod(filename, os.FileMode(header.Mode))
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	fmt.Printf("Source code for %s downloaded and extracted.\n", rp.PackageName)

	return nil
}

func (rp RizinPackage) downloadGit(artifactsPath string) error {
	gitProjectName := gitProjectNameFromURL(rp.PackageSource.URL)
	projectPath := filepath.Join(artifactsPath, gitProjectName)
	fi, err := os.Stat(projectPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && fi.IsDir() {
		repo, err := git.PlainOpen(projectPath)
		if err != nil {
			return err
		}

		tree, err := repo.Worktree()
		if err != nil {
			return err
		}

		err = runWithDotProgress(
			fmt.Sprintf("Updating %s source repository...", rp.PackageName),
			gitProgressDotInterval,
			func() error {
				return tree.Pull(&git.PullOptions{
					Progress:          nil,
					RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
				})
			},
		)
		if err == nil {
			fmt.Printf("Source repository for %s updated.\n", rp.PackageName)
			return nil
		}
		if err == git.NoErrAlreadyUpToDate {
			fmt.Printf("Source repository for %s is already up to date.\n", rp.PackageName)
			return nil
		}
		return err
	}

	err = runWithDotProgress(
		fmt.Sprintf("Cloning %s source repository...", rp.PackageName),
		gitProgressDotInterval,
		func() error {
			_, err := git.PlainClone(projectPath, false, &git.CloneOptions{
				URL:               rp.PackageSource.URL,
				Progress:          nil,
				RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			})
			return err
		},
	)
	if err != nil {
		return err
	}
	fmt.Printf("Source repository for %s downloaded.\n", rp.PackageName)
	return nil
}

// Download the source code of a package and extract it in the provided path
func (rp RizinPackage) Download(baseArtifactsPath string) error {
	err := rp.validateSource()
	if err != nil {
		return err
	}

	log.Printf("Downloading package %s... from '%s'", rp.PackageName, rp.PackageSource.URL)
	artifactsPath := rp.artifactsPath(baseArtifactsPath)
	err = os.MkdirAll(artifactsPath, os.FileMode(0755))
	if err != nil {
		return err
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

	if err := runCommandWithDotProgress(fmt.Sprintf("Configuring %s build...", rp.PackageName), cmd); err != nil {
		return err
	}

	cmd = exec.Command("meson", "compile", "-C", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	return runCommandWithDotProgress(fmt.Sprintf("Building %s...", rp.PackageName), cmd)
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
	if err := runCommandWithDotProgress(fmt.Sprintf("Configuring %s build...", rp.PackageName), cmd); err != nil {
		return err
	}

	cmd = exec.Command("cmake", "--build", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	return runCommandWithDotProgress(fmt.Sprintf("Building %s...", rp.PackageName), cmd)
}

func (rp RizinPackage) installMeson(site Site) ([]string, error) {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	cmd := exec.Command("meson", "install", "-C", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := runCommandWithDotProgress(fmt.Sprintf("Installing %s...", rp.PackageName), cmd); err != nil {
		return nil, err
	}

	cmd = exec.Command("meson", "introspect", "--installed", "build")
	cmd.Dir = srcPath
	cmd.Stderr = log.Writer()
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var data map[string]string
	err = json.Unmarshal([]byte(out), &data)
	if err != nil {
		return nil, err
	}

	var installed_files []string
	for _, v := range data {
		installed_files = append(installed_files, v)
	}
	return installed_files, nil
}

func (rp RizinPackage) installCMake(site Site) ([]string, error) {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	cmd := exec.Command("cmake", "--install", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := runCommandWithDotProgress(fmt.Sprintf("Installing %s...", rp.PackageName), cmd); err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Join(srcPath, "build", "install_manifest.txt"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func (rp RizinPackage) uninstallMeson(site Site) error {
	srcPath := rp.sourcePath(site.GetArtifactsDir())
	cmd := exec.Command("ninja", "uninstall", "-C", "build")
	cmd.Dir = srcPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	return runCommandWithDotProgress(fmt.Sprintf("Uninstalling %s...", rp.PackageName), cmd)
}

func buildErrorMsg(msg string) string {
	if runtime.GOOS == "windows" {
		return "To build Rizin packages on Windows you need to enable the 'Developer Command Prompt for Visual Studio'. Follow the instructions at https://learn.microsoft.com/en-us/visualstudio/ide/reference/command-prompt-powershell?view=vs-2022 to install and execute it. Moreover " + msg
	}
	return msg
}

// Build a package if a source is provided
func (rp RizinPackage) Build(site Site) error {
	if err := rp.validateSource(); err != nil {
		return err
	}

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

	switch rp.PackageSource.BuildSystem {
	case Meson:
		_, err := exec.LookPath("meson")
		if err != nil {
			return fmt.Errorf("%s", buildErrorMsg("make sure 'meson' is installed and in PATH"))
		}

		_, err = exec.LookPath("pkg-config")
		if err != nil {
			_, err = exec.LookPath("cmake")
			if err != nil {
				return fmt.Errorf("%s", buildErrorMsg("make sure either 'cmake' or `pkg-config` are installed and in PATH"))
			}
		}

		return rp.buildMeson(site)
	case CMake:
		_, err := exec.LookPath("cmake")
		if err != nil {
			return fmt.Errorf("%s", buildErrorMsg("make sure 'cmake' is installed and in PATH"))
		}

		_, err = exec.LookPath("pkg-config")
		if err != nil {
			return fmt.Errorf("%s", buildErrorMsg("make sure `pkg-config` is installed and in PATH"))
		}

		return rp.buildCMake(site)
	default:
		log.Printf("BuildSystem %s is not supported yet.", rp.PackageSource.BuildSystem)
		return fmt.Errorf("unsupported build system")
	}
}

// Install a package after building it
func (rp RizinPackage) Install(site Site) ([]string, error) {
	if err := rp.validateSource(); err != nil {
		return []string{}, err
	}

	err := rp.Build(site)
	if err != nil {
		return []string{}, err
	}

	var installed_files []string
	switch rp.PackageSource.BuildSystem {
	case Meson:
		installed_files, err = rp.installMeson(site)
	case CMake:
		installed_files, err = rp.installCMake(site)
	default:
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
	if err := rp.validateSource(); err != nil {
		return err
	}

	var err error
	switch rp.PackageSource.BuildSystem {
	case Meson:
		err = rp.uninstallMeson(site)
	default:
		log.Printf("BuildSystem %s is not supported yet.", rp.PackageSource.BuildSystem)
		err = fmt.Errorf("unsupported build system")
	}
	if err != nil {
		return err
	}
	fmt.Printf("Package %s uninstalled.\n", rp.PackageName)
	return nil
}
