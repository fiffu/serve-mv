package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

// Defaults
const (
	DefaultDirectory = "."
	DefaultDomain    = "serve-mv.local"
	DefaultSubdomain = ""
	DefaultPort      = 9001
)

func parseArgs(parsed *cli.Context) Options {
	return Options{
		Directory: parsed.String("dir"),
		Domain:    parsed.String("domain"),
		Subdomain: parsed.String("subdomain"),
		Port:      parsed.Int("port"),
	}
}

func main() {
	app := &cli.App{
		Name: "serve-mv",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dir",
				Value:       DefaultDirectory,
				Usage:       "serve this directory",
				DefaultText: "current directory",
			},
			&cli.StringFlag{
				Name:  "domain",
				Value: DefaultDomain,
				Usage: "the DNS domain to use",
			},
			&cli.StringFlag{
				Name:        "subdomain",
				Value:       DefaultSubdomain,
				Usage:       "the DNS subdomain to use",
				DefaultText: `generated using md5 of "gameTitle" from System.json`,
			},
			&cli.IntFlag{
				Name:  "port",
				Value: DefaultPort,
				Usage: "the network port to use",
			},
		},
		Action: func(c *cli.Context) error {
			opts := parseArgs(c)
			if err := Listen(opts); err != nil {
				return err
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
