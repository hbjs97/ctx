package main

import (
	"os"

	"github.com/hbjs97/ctx/internal/cli"
)

func main() {
	app := cli.NewApp()
	cmd := app.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(int(cli.MapExitCode(err)))
	}
}
