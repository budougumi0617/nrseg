package main

import (
	"os"

	"github.com/budougumi0617/nrseg"
)

var (
	Version  = "devel"
	Revision = "unset"
)

func main() {
	if err := nrseg.Run(os.Args, os.Stdout, os.Stderr, Version, Revision); err != nil {
		os.Exit(1)
	}
}
