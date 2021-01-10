package nrseg

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type nrseg struct {
	in, dist             string
	outStream, errStream io.Writer
}

func fill(args []string, outStream, errStream io.Writer) (*nrseg, error) {
	cn := args[0]
	flags := flag.NewFlagSet(cn, flag.ContinueOnError)
	flags.SetOutput(errStream)

	if err := flags.Parse(args[1:]); err != nil {
		return nil, err
	}

	dir := "./"
	nargs := flags.Args()
	if len(nargs) > 1 {
		msg := "execution path must be only one or no-set(current dirctory)."
		return nil, fmt.Errorf(msg)
	}
	if len(nargs) == 1 {
		dir = nargs[0]
	}

	// parse args
	return &nrseg{
		in:        dir,
		dist:      dir,
		outStream: outStream,
		errStream: errStream,
	}, nil
}

func (n *nrseg) run() error {
	return filepath.Walk(n.in, func(path string, info os.FileInfo, err error) error {
		return nil
	})
}

// Run is entry point.
func Run(args []string, outStream, errStream io.Writer) error {
	nrseg, err := fill(args, outStream, errStream)
	if err != nil {
		return err
	}
	return nrseg.run()
}
