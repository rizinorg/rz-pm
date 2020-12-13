package features

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/rizinorg/rz-pm/pkg/rzpackage"
	"github.com/rizinorg/rz-pm/pkg/site"
)

const (
	DebugEnvVar = "RZPM_DEBUG"

	msgCannotInitialize = "could not initialize: %w"
)

func Delete(rzpmDir string) error {
	s, err := site.New(rzpmDir)
	if err != nil {
		return fmt.Errorf(msgCannotInitialize, err)
	}

	return s.Remove()
}

func Init(rzpmDir string) error {
	s, err := site.New(rzpmDir)
	if err != nil {
		return fmt.Errorf("could not initialize: %w", err)
	}

	return s.Database().InitOrUpdate()
}

func Install(rzpmDir, packageName string) error {
	s, err := site.New(rzpmDir)
	if err != nil {
		return fmt.Errorf(msgCannotInitialize, err)
	}

	return s.InstallPackage(packageName)
}

func InstallFromFile(rzpmDir, path string) error {
	s, err := site.New(rzpmDir)
	if err != nil {
		return fmt.Errorf(msgCannotInitialize, err)
	}

	return s.InstallPackageFromFile(path)
}

func InstallRizin(rzpmDir, rzDir, version string) error {
	s, err := site.New(rzpmDir)
	if err != nil {
		return err
	}

	return s.InstallRizin(rzDir, version)
}

func UninstallRizin(rzpmDir, rzDir string) error {
	s, err := site.New(rzpmDir)
	if err != nil {
		return err
	}

	return s.UninstallRizin(rzDir)
}

func ListAvailable(rzpmDir string) ([]rzpackage.Info, error) {
	s, err := site.New(rzpmDir)
	if err != nil {
		return nil, fmt.Errorf(msgCannotInitialize, err)
	}

	return s.Database().ListAvailablePackages()
}

func ListInstalled(rzpmDir string) ([]rzpackage.Info, error) {
	s, err := site.New(rzpmDir)
	if err != nil {
		return nil, fmt.Errorf(msgCannotInitialize, err)
	}

	return s.ListInstalledPackages()
}

func Search(rzpmDir, pattern string) ([]rzpackage.Info, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("%q is not a valid regex: %w", pattern, err)
	}

	packages, err := ListAvailable(rzpmDir)
	if err != nil {
		return nil, fmt.Errorf("could not get the list of packages: %w", err)
	}

	matches := make([]rzpackage.Info, 0, len(packages))

	for _, p := range packages {
		if re.Match([]byte(p.Name)) {
			matches = append(matches, p)
		}
	}

	return matches, nil
}

func SetDebug(value bool) {
	if value {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(ioutil.Discard)
	}
}

func Uninstall(rzpmDir, packageName string) error {
	s, err := site.New(rzpmDir)
	if err != nil {
		return fmt.Errorf(msgCannotInitialize, err)
	}

	return s.UninstallPackage(packageName)
}

func Upgrade(rzpmDir, packageName string) error {
	s, err := site.New(rzpmDir)
	if err != nil {
		return fmt.Errorf(msgCannotInitialize, err)
	}

	return s.Upgrade(packageName)
}

func UpgradeAll(rzpmDir string) error {
	s, err := site.New(rzpmDir)
	if err != nil {
		return fmt.Errorf(msgCannotInitialize, err)
	}

	packages, err := s.ListInstalledPackages()
	if err != nil {
		log.Print(err)
		return errors.New("could not list the installed packages")
	}

	failed := make([]string, 0, len(packages))

	for _, p := range packages {
		name := p.Name

		log.Println("Upgrading " + name)

		if err := s.Upgrade(name); err != nil {
			log.Print(err)
			failed = append(failed, name)
		}
	}

	sort.Strings(failed)

	if len(failed) > 0 {
		return fmt.Errorf(
			"could not upgrade the following packages: %s",
			strings.Join(failed, ", "))
	}

	return nil
}
