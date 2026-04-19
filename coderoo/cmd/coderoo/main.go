package main

import (
	"os"

	"github.com/coderoo-dev/coderoo/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
