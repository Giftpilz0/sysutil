package main

import (
	"os"

	"github.com/giftpilz0/sysutil/cmd"
)

// version is set at build time by goreleaser.
var version = "dev"

func main() {
	os.Exit(cmd.Execute(version))
}
