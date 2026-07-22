package main

import (
	"os"

	"github.com/thaodangspace/things-cli/cli"
)

func main() {
	os.Exit(cli.Execute())
}
