package main

import (
	"flag"
	"fmt"
	"testing"

	rzpmPkg "github.com/rizinorg/rz-pm/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

type fakeCLIPackage struct {
	name string
}

func (p fakeCLIPackage) Name() string                           { return p.name }
func (p fakeCLIPackage) Version() string                        { return "0.0.1" }
func (p fakeCLIPackage) Summary() string                        { return "" }
func (p fakeCLIPackage) Description() string                    { return "" }
func (p fakeCLIPackage) Source() rzpmPkg.RizinPackageSource     { return rzpmPkg.RizinPackageSource{} }
func (p fakeCLIPackage) Download(string) error                  { return nil }
func (p fakeCLIPackage) Build(rzpmPkg.Site) error               { return nil }
func (p fakeCLIPackage) Install(rzpmPkg.Site) ([]string, error) { return nil, nil }
func (p fakeCLIPackage) Uninstall(rzpmPkg.Site) error           { return nil }

type fakeCLISite struct {
	packages        map[string]rzpmPkg.Package
	installCalls    []string
	uninstallCalls  []string
	cleanCalls      []string
	closeCalls      int
	getPackageCalls []string
}

func (s *fakeCLISite) ListAvailablePackages() ([]rzpmPkg.Package, error) {
	return []rzpmPkg.Package{}, nil
}
func (s *fakeCLISite) ListInstalledPackages() ([]rzpmPkg.Package, error) {
	return []rzpmPkg.Package{}, nil
}
func (s *fakeCLISite) IsPackageInstalled(rzpmPkg.Package) bool { return false }
func (s *fakeCLISite) GetPackage(name string) (rzpmPkg.Package, error) {
	s.getPackageCalls = append(s.getPackageCalls, name)
	pkg, ok := s.packages[name]
	if !ok {
		return nil, fmt.Errorf("package %s not found", name)
	}
	return pkg, nil
}
func (s *fakeCLISite) GetPackageFromFile(string) (rzpmPkg.Package, error) { return nil, nil }
func (s *fakeCLISite) GetInstalledPackage(string) (rzpmPkg.InstalledPackage, error) {
	return rzpmPkg.InstalledPackage{}, nil
}
func (s *fakeCLISite) GetBaseDir() string      { return "" }
func (s *fakeCLISite) GetArtifactsDir() string { return "" }
func (s *fakeCLISite) GetPkgConfigDir() string { return "" }
func (s *fakeCLISite) GetCMakeDir() string     { return "" }
func (s *fakeCLISite) InstallPackage(pkg rzpmPkg.Package) error {
	s.installCalls = append(s.installCalls, pkg.Name())
	return nil
}
func (s *fakeCLISite) UninstallPackage(pkg rzpmPkg.Package) error {
	s.uninstallCalls = append(s.uninstallCalls, pkg.Name())
	return nil
}
func (s *fakeCLISite) CleanPackage(pkg rzpmPkg.Package) error {
	s.cleanCalls = append(s.cleanCalls, pkg.Name())
	return nil
}
func (s *fakeCLISite) Remove() error        { return nil }
func (s *fakeCLISite) RizinVersion() string { return "0.9.0" }
func (s *fakeCLISite) Close() error         { s.closeCalls++; return nil }

func newCLIContext(t *testing.T, args []string, includeClean bool) *cli.Context {
	t.Helper()

	flagSet := flag.NewFlagSet("rz-pm-test", flag.ContinueOnError)
	flagSet.Bool(flagUpdateDB, true, "")
	flagSet.Bool("file", false, "")
	if includeClean {
		flagSet.Bool("clean", false, "")
	}
	require.NoError(t, flagSet.Parse(args))

	app := cli.NewApp()
	return cli.NewContext(app, flagSet, nil)
}

func TestInstallPackagesUsesSingleSite(t *testing.T) {
	originalInitSite := initSite
	defer func() { initSite = originalInitSite }()

	site := &fakeCLISite{
		packages: map[string]rzpmPkg.Package{
			"first":  fakeCLIPackage{name: "first"},
			"second": fakeCLIPackage{name: "second"},
		},
	}
	initCalls := 0
	initSite = func(string, bool) (rzpmPkg.Site, error) {
		initCalls++
		return site, nil
	}

	//reusing same site for whole cmd, where it's multi-pkg
	err := installPackages(newCLIContext(t, []string{"--clean", "first", "second"}, true))
	require.NoError(t, err)

	assert.Equal(t, 1, initCalls, "site should be initialized once for a multi-package install")
	assert.Equal(t, 1, site.closeCalls, "site should be closed once")
	assert.Equal(t, []string{"first", "second"}, site.getPackageCalls)
	assert.Equal(t, []string{"first", "second"}, site.cleanCalls)
	assert.Equal(t, []string{"first", "second"}, site.installCalls)
}

func TestUninstallPackagesUsesSingleSite(t *testing.T) {
	originalInitSite := initSite
	defer func() { initSite = originalInitSite }()

	site := &fakeCLISite{
		packages: map[string]rzpmPkg.Package{
			"first":  fakeCLIPackage{name: "first"},
			"second": fakeCLIPackage{name: "second"},
		},
	}
	initCalls := 0
	initSite = func(string, bool) (rzpmPkg.Site, error) {
		initCalls++
		return site, nil
	}

	//same as in install above
	err := uninstallPackages(newCLIContext(t, []string{"first", "second"}, false))
	require.NoError(t, err)

	assert.Equal(t, 1, initCalls, "site should be initialized once for a multi-package uninstall")
	assert.Equal(t, 1, site.closeCalls, "site should be closed once")
	assert.Equal(t, []string{"first", "second"}, site.getPackageCalls)
	assert.Equal(t, []string{"first", "second"}, site.uninstallCalls)
}
