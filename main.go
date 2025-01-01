package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
)

// Defaults
const (
	DefaultDirectory = "."
	DefaultPort      = 9001
)

func parseArgs(parsed *cli.Context) Options {
	return Options{
		Directory: parsed.String("dir"),
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
			&cli.IntFlag{
				Name:  "port",
				Value: DefaultPort,
				Usage: "the network port to use",
			},
		},
		Action: func(c *cli.Context) error {
			opts := parseArgs(c)
			return Listen(opts)
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c

		Server.PrintReport()
		os.Exit(0)
	}()

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
