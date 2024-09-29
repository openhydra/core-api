package main

import (
	"os"

	"core-api/cmd/core-api-server/app"
)

var version string

func main() {
	cmd := app.NewCommand(version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
