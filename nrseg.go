package nrseg

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ErrShowVersion returns when set version flag.
	ErrShowVersion = errors.New("show version")
)

var (
	Version  = "devel"
	Revision = "unset"
)

type nrseg struct {
	in, dist             string
	ignoreDirs           []string
	outStream, errStream io.Writer
}

func fill(args []string, outStream, errStream io.Writer) (*nrseg, error) {
	cn := args[0]
	flags := flag.NewFlagSet(cn, flag.ContinueOnError)
	flags.SetOutput(errStream)
	flags.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"Insert function segments into any function/method for Newrelic APM.\n\nUsage of %s:\n",
			os.Args[0],
		)
		flags.PrintDefaults()
	}

	var v bool
	vdesc := "print version information and quit."
	flags.BoolVar(&v, "version", false, vdesc)
	flags.BoolVar(&v, "v", false, vdesc)

	var ignoreDirs string
	idesc := "ignore directory names. ex: foo,bar,baz"
	flags.StringVar(&ignoreDirs, "ignore", "", idesc)
	flags.StringVar(&ignoreDirs, "i", "", idesc)

	if err := flags.Parse(args[1:]); err != nil {
		return nil, err
	}
	if v {
		fmt.Fprintf(errStream, "%s version %q, revison %q\n", cn, Version, Revision)
		return nil, ErrShowVersion
	}

	dirs := []string{"testdata"}
	if len(ignoreDirs) != 0 {
		dirs = append(dirs, strings.Split(ignoreDirs, ",")...)
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
		in:         dir,
		ignoreDirs: dirs,
		outStream:  outStream,
		errStream:  errStream,
	}, nil
}

var c = regexp.MustCompile("(?m)^// Code generated .* DO NOT EDIT\\.$")

func (n *nrseg) skipDir(p string) bool {
	for _, dir := range n.ignoreDirs {
		if filepath.Base(p) == dir {
			return true
		}
	}
	return false
}

func (n *nrseg) run() error {
	return filepath.Walk(n.in, func(path string, info os.FileInfo, err error) error {
		fmt.Fprintf(n.outStream, "walk %q\n", path)
		if info.IsDir() && n.skipDir(path) {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			return nil
		}

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
			fmt.Fprintf(n.errStream, "process failed %q: %v\n", path, err)
			return err
		}
		fmt.Fprintf(n.outStream, "got %q\n", got)
		if !bytes.Equal(org, got) {
			fmt.Fprintf(n.outStream, "update!! %q\n", path)
			if len(n.dist) != 0 && n.in != n.dist {
				fmt.Fprintf(n.outStream, "update!! %q\n", n.dist)
				return n.writeOtherPath(n.in, n.dist, path, got)
			}
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
	distabs, err := filepath.Abs(dist)
	if err != nil {
		return err
	}
	dp := filepath.Join(distabs, p)
	dpd := filepath.Dir(dp)
	if _, err := os.Stat(dpd); os.IsNotExist(err) {
		if err := os.Mkdir(dpd, 0777); err != nil {
			fmt.Fprintf(n.outStream, "create dir failed at %q: %v\n", dpd, err)
			return err
		}
	}

	fmt.Fprintf(n.outStream, "update file %q\n", dp)
	f, err := os.OpenFile(dp, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil
	}
	defer f.Close()
	_, err = f.Write(got)
	if err != nil {
		fmt.Fprintf(n.outStream, "write file failed %v\n", err)
	}
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
