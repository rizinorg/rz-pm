package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

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
			ArgsUsage: "[PACKAGE]",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "f",
					Usage: "install a package described by a local file",
				},
			},
			Action: func(c *cli.Context) error {
				return fmt.Errorf("install command is not implemented yet")
			},
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "list packages",
			Action: func(c *cli.Context) error {
				return fmt.Errorf("list command is not implemented yet")
			},
		},
		{
			Name:      "search",
			Usage:     "search for a package in the database",
			ArgsUsage: "PATTERN",
			Action: func(c *cli.Context) error {
				return fmt.Errorf("search command is not implemented yet")
			},
		},
		{
			Name:      "uninstall",
			Usage:     "uninstall a package",
			ArgsUsage: "PACKAGE",
			Action: func(c *cli.Context) error {
				return fmt.Errorf("uninstall command is not implemented yet")
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
