package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

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
	app.Version = "0.1.5"

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
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
