package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/fatih/color"
	"github.com/hashicorp/go-version"
	"github.com/inconshreveable/go-update"
	"github.com/rizinorg/rz-pm/pkg"
	"github.com/rizinorg/rz-pm/pkg/updatecheck"
	"github.com/urfave/cli/v2"
)

const (
	debugEnvVar     = "RZPM_DEBUG"
	flagNameDebug   = "debug"
	flagSkipUpgrade = "skip-upgrade"
	flagUpdateDB    = "update-db"

	updateDBCheckFile = "last_db_update_check"
	updateCheckFile   = "last_update_check"
)

var (
	updateChecker = updatecheck.Checker{
		Path:     filepath.Join(xdg.CacheHome, "rz-pm", updateCheckFile),
		Interval: 24 * time.Hour,
	}
	dbUpdateChecker = updatecheck.Checker{
		Path:     filepath.Join(xdg.CacheHome, "rz-pm", updateDBCheckFile),
		Interval: 24 * time.Hour,
	}
)

func setDebug(value bool) {
	if value {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(io.Discard)
	}
}

func listPackages(c *cli.Context, installed bool) error {
	if c.Args().Len() != 0 {
		cli.ShowCommandHelp(c, "list")
		return fmt.Errorf("wrong usage of list command")
	}

	site, err := pkg.InitSite(pkg.SiteDir(), c.Bool(flagUpdateDB))
	if err != nil {
		return err
	}
	defer site.Close()

	var packages []pkg.Package
	if installed {
		packages, err = site.ListInstalledPackages()
	} else {
		packages, err = site.ListAvailablePackages()
	}
	if err != nil {
		return err
	}

	green := color.New(color.Bold, color.FgGreen).SprintFunc()
	red := color.New(color.Bold, color.FgRed).SprintFunc()

	for _, myPkg := range packages {
		info := ""
		if site.IsPackageInstalled(myPkg) {
			info = green(" [installed]")

			installedPackage, err := site.GetInstalledPackage(myPkg.Name())
			if err == nil && installedPackage.RizinVersion != nil {
				if pkg.GetMajorMinorVersion(site.RizinVersion()) != *installedPackage.RizinVersion {
					info += red(fmt.Sprintf(" [for rizin v%s]", *installedPackage.RizinVersion))
				}
			}
		}

		fmt.Printf("%s: %s%s\n", myPkg.Name(), myPkg.Summary(), info)
	}
	return nil
}

func listAvailablePackages(c *cli.Context) error {
	return listPackages(c, false)
}

func listInstalledPackages(c *cli.Context) error {
	return listPackages(c, true)
}

func infoPackage(c *cli.Context) error {
	packageName := c.Args().First()
	if packageName == "" || c.Args().Len() != 1 {
		cli.ShowCommandHelp(c, "info")
		return fmt.Errorf("wrong usage of info command")
	}

	site, err := pkg.InitSite(pkg.SiteDir(), c.Bool(flagUpdateDB))
	if err != nil {
		return err
	}
	defer site.Close()

	var pkg pkg.Package
	if c.Bool("file") {
		pkg, err = site.GetPackageFromFile(packageName)
	} else {
		pkg, err = site.GetPackage(packageName)
	}
	if err != nil {
		return err
	}

	var isInstalled string
	if site.IsPackageInstalled(pkg) {
		isInstalled = "yes"
	} else {
		isInstalled = "no"
	}

	fmt.Printf("Name: %s\n", pkg.Name())
	fmt.Printf("Version: %s\n", pkg.Version())
	fmt.Printf("Summary: %s\n", pkg.Summary())
	fmt.Printf("Description: %s\n", pkg.Description())
	fmt.Printf("Installed: %s\n", isInstalled)
	return nil
}

func getNewRzPmVersion() (*version.Version, error) {
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("https://github.com/rizinorg/rz-pm/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 302 {
		return nil, fmt.Errorf("expected a redirection when querying releases/latest URL")
	}
	redirect_url, err := resp.Location()
	if err != nil {
		return nil, err
	}
	redirect_url_parts := strings.Split(redirect_url.Path, "/")
	new_version_str := redirect_url_parts[len(redirect_url_parts)-1]

	return version.NewVersion(new_version_str)
}

func getRzPmName() string {
	name := "rz-pm-" + runtime.GOOS + "-"
	if runtime.GOARCH == "amd64" {
		name += "x86_64"
	} else {
		name += runtime.GOARCH
	}
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func checkUpgrade(c *cli.Context) (bool, *version.Version, *version.Version, error) {
	new_version, err := getNewRzPmVersion()
	if err != nil {
		return false, nil, nil, err
	}

	current_version, err := version.NewVersion(c.App.Version)
	if err != nil {
		return false, nil, nil, err
	}

	if new_version.LessThanOrEqual(current_version) {
		return false, nil, nil, nil
	}
	return true, current_version, new_version, nil
}

func upgradeRzPm(c *cli.Context) error {
	needsUpgrade, current_version, new_version, err := checkUpgrade(c)
	if err != nil {
		return nil
	} else if !needsUpgrade {
		fmt.Printf("You are already on the latest rz-pm version!\n")
		return nil
	}

	fmt.Println("Your version of rz-pm is not the latest one.")
	fmt.Printf("Currently installed version: %s, available version: %s\n", current_version, new_version)

	fmt.Println("Downloading the new version...")
	client := http.Client{}
	rz_pm_name := getRzPmName()
	resp, err := client.Get("https://github.com/rizinorg/rz-pm/releases/latest/download/" + rz_pm_name)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = update.Apply(resp.Body, update.Options{})
	if err != nil {
		return err
	}
	fmt.Printf("Upgrade to rz-pm version %s was successful!\n", new_version)
	return nil
}

func installPackages(c *cli.Context) error {
	if c.Args().Len() < 1 {
		cli.ShowCommandHelp(c, "install")
		return fmt.Errorf("wrong usage of install command")
	}
	for _, packageName := range c.Args().Slice() {
		if packageName == "" {
			cli.ShowCommandHelp(c, "install")
			return fmt.Errorf("wrong usage of install command")
		}
		site, err := pkg.InitSite(pkg.SiteDir(), c.Bool(flagUpdateDB))
		if err != nil {
			return err
		}
		defer site.Close()

		var pkg pkg.Package
		if c.Bool("file") {
			pkg, err = site.GetPackageFromFile(packageName)
		} else {
			pkg, err = site.GetPackage(packageName)
		}
		if err != nil {
			return err
		}

		if c.Bool("clean") {
			site.CleanPackage(pkg)
		}

		err = site.InstallPackage(pkg)
		if err != nil {
			return err
		}
	}
	return nil
}

func uninstallPackages(c *cli.Context) error {
	if c.Args().Len() < 1 {
		cli.ShowCommandHelp(c, "uninstall")
		return fmt.Errorf("wrong usage of uninstall command")
	}
	for _, packageName := range c.Args().Slice() {
		if packageName == "" {
			cli.ShowCommandHelp(c, "uninstall")
			return fmt.Errorf("wrong usage of uninstall command")
		}

		site, err := pkg.InitSite(pkg.SiteDir(), c.Bool(flagUpdateDB))
		if err != nil {
			return err
		}
		defer site.Close()

		var pkg pkg.Package
		if c.Bool("file") {
			pkg, err = site.GetPackageFromFile(packageName)
		} else {
			pkg, err = site.GetPackage(packageName)
		}
		if err != nil {
			return err
		}

		err = site.UninstallPackage(pkg)
		if err != nil {
			return err
		}
	}
	return nil
}

func cleanPackage(c *cli.Context) error {
	packageName := c.Args().First()
	if packageName == "" || c.Args().Len() != 1 {
		cli.ShowCommandHelp(c, "clean")
		return fmt.Errorf("wrong usage of clean command")
	}

	site, err := pkg.InitSite(pkg.SiteDir(), c.Bool(flagUpdateDB))
	if err != nil {
		return err
	}
	defer site.Close()

	var pkg pkg.Package
	if c.Bool("file") {
		pkg, err = site.GetPackageFromFile(packageName)
	} else {
		pkg, err = site.GetPackage(packageName)
	}
	if err != nil {
		return err
	}

	err = site.CleanPackage(pkg)
	if err != nil {
		return err
	}
	fmt.Printf("Package %s build artifacts have been cleaned.\n", pkg.Name())
	return nil
}

func checkNewerVersionOnline(c *cli.Context) error {
	setDebug(c.Bool(flagNameDebug))

	if !c.Bool(flagSkipUpgrade) && c.Args().First() != "upgrade" {
		shouldCheck, err := updateChecker.ShouldCheck()
		if err != nil {
			// If we can't determine, default to checking
			shouldCheck = true
		}

		if shouldCheck {
			log.Println("Checking for newer rz-pm version online...")
			needsUpgrade, current_version, new_version, err := checkUpgrade(c)
			if err != nil {
				return nil
			} else if !needsUpgrade {
				_ = updateChecker.UpdateTimestamp()
				return nil
			}

			fmt.Println("Your version of rz-pm is not the latest one.")
			fmt.Printf("Currently installed version: %s, available version: %s\n", current_version, new_version)
			fmt.Println()
			fmt.Println("Run the 'upgrade' command to upgrade rz-pm.")
			_ = updateChecker.UpdateTimestamp()
			os.Exit(0)
		} else {
			log.Println("Skipping update check, last check was recent enough.")
		}
	}

	shouldUpdateDB, err := dbUpdateChecker.ShouldCheck()
	if err != nil {
		// If we can't determine, default to checking
		shouldUpdateDB = true
	}

	if shouldUpdateDB {
		log.Println("We didn't check the DB for updates for a while, checking now...")
		c.Set(flagUpdateDB, "true")
		dbUpdateChecker.UpdateTimestamp()
	} else {
		log.Println("Skipping DB update check, last check was recent enough.")
	}

	return nil
}

func main() {

	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"V"},
		Usage:   "print only the version",
	}

	app := cli.NewApp()
	app.Name = "rz-pm"
	app.Usage = "Rizin package manager"
	app.Version = "v0.3.3"

	cli.AppHelpTemplate = fmt.Sprintf(`%s
RZ_PM_SITE:
   %s
`, cli.AppHelpTemplate, pkg.SiteDir())

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    flagNameDebug,
			Usage:   "enable debug logs",
			EnvVars: []string{debugEnvVar},
		},
		&cli.BoolFlag{
			Name:  flagSkipUpgrade,
			Usage: "skip auto-upgrade on start",
		},
		&cli.BoolFlag{
			Name:  flagUpdateDB,
			Usage: "Update the DB?",
			Value: false,
		},
	}

	app.Before = checkNewerVersionOnline

	app.Commands = []*cli.Command{
		{
			Name:      "install",
			Usage:     "install a package",
			ArgsUsage: "<package-name> [<package-name> ...]",
			Action:    installPackages,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "clean",
					Usage: "do a clean before installing the package",
				},
				&cli.BoolFlag{
					Name:  "file",
					Usage: "install a local file(s)",
				},
			},
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "list packages",
			Action:  listAvailablePackages,
			Subcommands: []*cli.Command{
				{
					Name:   "available",
					Usage:  "list all available packages",
					Action: listAvailablePackages,
				},
				{
					Name:   "installed",
					Usage:  "list installed packages",
					Action: listInstalledPackages,
				},
			},
		},
		{
			Name:      "uninstall",
			Usage:     "uninstall a package",
			ArgsUsage: "<package-name> [<package-name> ...]",
			Action:    uninstallPackages,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "file",
					Usage: "info a local file",
				},
			},
		},
		{
			Name:      "clean",
			Usage:     "remove any temporary build artifacts of a package",
			ArgsUsage: "<package-name>",
			Action:    cleanPackage,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "file",
					Usage: "info a local file",
				},
			},
		},
		{
			Name:   "info",
			Usage:  "info about a package",
			Action: infoPackage,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "file",
					Usage: "info a local file",
				},
			},
		},
		{
			Name:   "upgrade",
			Usage:  "upgrade rz-pm",
			Action: upgradeRzPm,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
