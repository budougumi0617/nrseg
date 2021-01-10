package nrseg

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

	return &nrseg{
		in:        dir,
		dist:      "./tmp",
		outStream: outStream,
		errStream: errStream,
	}, nil
}

func (n *nrseg) run() error {
	return filepath.Walk(n.in, func(path string, info os.FileInfo, err error) error {
		fmt.Fprintf(n.outStream, "walk %q\n", path)
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		// TODO: ignore auto generated file.
		fmt.Fprintf(n.outStream, "start %q\n", path)
		f, err := os.OpenFile(path, os.O_RDWR, 0664)
		if err != nil {
			fmt.Fprintf(n.errStream, "cannot open %q: %v\n", path, err)
			return err
		}
		defer f.Close()
		org, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Fprintf(n.errStream, "cannot read %q: %v\n", path, err)
			return err
		}
		got, err := Process(path, org)
		if err != nil {
			fmt.Fprintf(n.errStream, "Process failed %q: %v\n", path, err)
			return err
		}
		fmt.Fprintf(n.outStream, "got %q\n", got)
		if !bytes.Equal(org, got) {
			fmt.Fprintf(n.outStream, "update!! %q\n", path)
			// for test
			//if n.in != n.dist {
			//	fmt.Fprintf(n.outStream, "update!! %q\n", n.dist)
			//	return n.writeOtherPath(n.in, n.dist, path, got)
			//}
			if _, err := f.WriteAt(got, 0); err != nil {
				fmt.Fprintf(n.errStream, "file update failed %q: %v\n", path, err)
				return err
			}
		}
		return nil
	})
}

func (n *nrseg) writeOtherPath(in, dist, path string, got []byte) error {
	p, err := filepath.Rel(in, path)
	if err != nil {
		return err
	}
	dp := filepath.Join(dist, p)
	fmt.Fprintf(n.outStream, "update file %q\n", dp)
	f, err := os.OpenFile(dp, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil
	}
	defer f.Close()
	_, err = f.Write(got)
	fmt.Printf("created at %q\n", dp)
	return err
}

// Run is entry point.
func Run(args []string, outStream, errStream io.Writer) error {
	nrseg, err := fill(args, outStream, errStream)
	if err != nil {
		return err
	}
	return nrseg.run()
}
