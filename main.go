package main

import (
	"os"

	"github.com/ikafly144/gobot/pkg/cli"
)

func main() {
	defer os.Exit(0)
	cli.Run()
}
