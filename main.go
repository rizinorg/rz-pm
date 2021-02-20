package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/rizinorg/rz-pm/internal/features"
	"github.com/rizinorg/rz-pm/internal/util/dir"
	"github.com/rizinorg/rz-pm/pkg/rzpackage"
)

func getArgumentOrExit(c *cli.Context) string {
	packageName := c.Args().First()

	if packageName == "" {
		if err := cli.ShowSubcommandHelp(c); err != nil {
			log.Fatal(err)
		}

		os.Exit(1)
	}

	return packageName
}

func main() {
	rzDir := dir.RzDir()
	rzpmDir := dir.SiteDir()

	listAvailablePackages := func(c *cli.Context) error {
		packages, err := features.ListAvailable(rzpmDir)
		if err != nil {
			return err
		}

		fmt.Printf("%d available packages\n", len(packages))
		printPackageSlice(packages)

		return nil
	}

	const flagNameDebug = "debug"

	app := cli.NewApp()
	app.Name = "rz-pm"
	app.Usage = "rizin package manager"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    flagNameDebug,
			Usage:   "enable debug logs",
			EnvVars: []string{features.DebugEnvVar},
		},
	}

	app.Before = func(c *cli.Context) error {
		features.SetDebug(c.Bool(flagNameDebug))
		return nil
	}

	app.Commands = []*cli.Command{
		{
			Name:  "delete",
			Usage: "delete the local package database",
			Action: func(*cli.Context) error {
				return features.Delete(rzpmDir)
			},
		},
		{
			Name:    "init",
			Aliases: []string{"update"},
			Usage:   "initialize or update the local package database",
			Action: func(*cli.Context) error {
				return features.Init(rzpmDir)
			},
		},
		{
			Name:      "install",
			Usage:     "install a package",
			ArgsUsage: "[PACKAGE]",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "f",
					Usage: "install a package described by a local file",
				},
			},
			Action: func(c *cli.Context) error {
				if path := c.String("f"); path != "" {
					log.Print("Installing " + path)
					return features.InstallFromFile(rzpmDir, path)
				}

				packageName := getArgumentOrExit(c)

				return features.Install(rzpmDir, packageName)
			},
			Subcommands: []*cli.Command{
				{
					Name:      "rizin",
					Usage:     "install rizin",
					ArgsUsage: "VERSION",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "p",
							Usage: "rizin's configure --prefix",
							Value: rzDir,
						},
					},
					Action: func(c *cli.Context) error {
						if c.NArg() != 1 {
							return errors.New("a version number is required")
						}

						version := c.Args().First()

						prefix := c.String("p")
						if prefix == "" {
							return errors.New("A prefix is required")
						}

						return features.InstallRizin(rzpmDir, rzDir, version)
					},
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
					Usage:  "list all the available packages",
					Action: listAvailablePackages,
				},
				{
					Name:  "installed",
					Usage: "list all the installed packages",
					Action: func(c *cli.Context) error {
						packages, err := features.ListInstalled(rzpmDir)
						if err != nil {
							return err
						}

						fmt.Printf("%d installed packages\n", len(packages))
						printPackageSlice(packages)

						return nil
					},
				},
			},
		},
		{
			Name:      "search",
			Usage:     "search for a package in the database",
			ArgsUsage: "PATTERN",
			Action: func(c *cli.Context) error {
				pattern := getArgumentOrExit(c)

				matches, err := features.Search(rzpmDir, pattern)
				if err != nil {
					return err
				}

				fmt.Printf("Your search returned %d matches\n", len(matches))
				printPackageSlice(matches)

				return nil
			},
		},
		{
			Name:      "uninstall",
			Usage:     "uninstall a package",
			ArgsUsage: "PACKAGE",
			Action: func(c *cli.Context) error {
				packageName := getArgumentOrExit(c)

				return features.Uninstall(rzpmDir, packageName)
			},
			Subcommands: []*cli.Command{
				{
					Name:  "rizin",
					Usage: "uninstall rizin",
					Action: func(c *cli.Context) error {
						return features.UninstallRizin(rzpmDir, rzDir)
					},
				},
			},
		},
		{
			Name:      "upgrade",
			Usage:     "upgrade (uninstall and reinstall) a package",
			ArgsUsage: "[PACKAGE]",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "a, all",
					Usage: "upgrade all packages",
				},
			},
			Action: func(c *cli.Context) error {
				if c.Bool("a") {
					return features.UpgradeAll(rzpmDir)
				}

				packageName := getArgumentOrExit(c)

				return features.Upgrade(rzpmDir, packageName)
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}

func printPackageSlice(packages []rzpackage.Info) {
	for _, p := range packages {
		fmt.Printf("%s: %s\n", p.Name, p.Desc)
	}
}
