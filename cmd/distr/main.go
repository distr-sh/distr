package main

import (
	"os"

	"github.com/distr-sh/distr/cmd/distr/cmd"
)

func main() {
	if err := cmd.RootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
