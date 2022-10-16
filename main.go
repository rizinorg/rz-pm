package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/inconshreveable/go-update"
	"github.com/rizinorg/rz-pm/pkg"
	"github.com/urfave/cli/v2"
)

const debugEnvVar = "RZPM_DEBUG"

func setDebug(value bool) {
	if value {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(ioutil.Discard)
	}
}

func listPackages(c *cli.Context, installed bool) error {
	if c.Args().Len() != 0 {
		cli.ShowCommandHelp(c, "list")
		return fmt.Errorf("wrong usage of list command")
	}

	site, err := pkg.InitSite(pkg.SiteDir())
	if err != nil {
		return err
	}
	var packages []pkg.Package
	if installed {
		packages, err = site.ListInstalledPackages()
	} else {
		packages, err = site.ListAvailablePackages()
	}
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		info := ""
		if site.IsPackageInstalled(pkg) {
			info = " [installed]"
		}
		fmt.Printf("%s: %s%s\n", pkg.Name(), pkg.Description(), info)
	}
	return nil
}

func listAvailablePackages(c *cli.Context) error {
	return listPackages(c, false)
}

func listInstalledPackages(c *cli.Context) error {
	return listPackages(c, true)
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
	return name
}

func upgradeRzPm(c *cli.Context) error {
	new_version, err := getNewRzPmVersion()
	if err != nil {
		return err
	}

	current_version, err := version.NewVersion(c.App.Version)
	if err != nil {
		return err
	}

	if new_version.LessThanOrEqual(current_version) {
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

func main() {
	const flagNameDebug = "debug"

	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"V"},
		Usage:   "print only the version",
	}

	app := cli.NewApp()
	app.Name = "rz-pm"
	app.Usage = "rizin package manager"
	app.Version = "v0.1.4"

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    flagNameDebug,
			Usage:   "enable debug logs",
			EnvVars: []string{debugEnvVar},
		},
	}

	app.Before = func(c *cli.Context) error {
		setDebug(c.Bool(flagNameDebug))
		return nil
	}

	app.Commands = []*cli.Command{
		{
			Name:      "install",
			Usage:     "install a package",
			ArgsUsage: "[package-name]",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "f",
					Usage: "install a package described by a local file",
				},
			},
			Action: func(c *cli.Context) error {
				packageName := c.Args().First()
				if packageName == "" || c.Args().Len() != 1 {
					cli.ShowCommandHelp(c, "install")
					return fmt.Errorf("wrong usage of install command")
				}

				site, err := pkg.InitSite(pkg.SiteDir())
				if err != nil {
					return err
				}

				pkg, err := site.GetPackage(packageName)
				if err != nil {
					return err
				}

				err = site.InstallPackage(pkg)
				if err != nil {
					return err
				}
				return nil
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
			ArgsUsage: "PACKAGE",
			Action: func(c *cli.Context) error {
				packageName := c.Args().First()
				if packageName == "" || c.Args().Len() != 1 {
					cli.ShowCommandHelp(c, "uninstall")
					return fmt.Errorf("wrong usage of uninstall command")
				}

				site, err := pkg.InitSite(pkg.SiteDir())
				if err != nil {
					return err
				}

				pkg, err := site.GetPackage(packageName)
				if err != nil {
					return err
				}

				err = site.UninstallPackage(pkg)
				if err != nil {
					return err
				}
				return nil
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
