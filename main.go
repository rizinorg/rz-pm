package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/rizinorg/rz-pm/internal/util/dir"
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

func main() {
	const flagNameDebug = "debug"

	app := cli.NewApp()
	app.Name = "rz-pm"
	app.Usage = "rizin package manager"
	app.Version = "0.0.1"

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
				if packageName == "" {
					cli.ShowCommandHelp(c, "install")
					return fmt.Errorf("wrong usage of install command")
				}

				site, err := pkg.InitSite(dir.SiteDir())
				if err != nil {
					return err
				}

				pkg, err := site.GetPackage(packageName)
				if err != nil {
					return err
				}

				err = site.DownloadPackage(pkg)
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
			Action: func(c *cli.Context) error {
				site, err := pkg.InitSite(dir.SiteDir())
				if err != nil {
					return err
				}
				packages, err := site.ListAvailablePackages()
				if err != nil {
					return err
				}

				for _, pkg := range packages {
					fmt.Printf("%s: %s\n", pkg.Name, pkg.Description)
				}
				return nil
			},
		},
		{
			Name:      "uninstall",
			Usage:     "uninstall a package",
			ArgsUsage: "PACKAGE",
			Action: func(c *cli.Context) error {
				packageName := c.Args().First()
				if packageName == "" {
					cli.ShowCommandHelp(c, "install")
					return fmt.Errorf("wrong usage of install command")
				}

				site, err := pkg.InitSite(dir.SiteDir())
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
