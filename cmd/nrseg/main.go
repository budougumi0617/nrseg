package main

import (
	"os"

	"github.com/budougumi0617/nrseg"
)

func main() {
	if err := nrseg.Run(os.Args, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
