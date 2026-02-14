package main

import (
	"os"

	"github.com/hbjs97/ctx/internal/cli"
)

func main() {
	cmd := cli.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(int(cli.MapExitCode(err)))
	}
}
