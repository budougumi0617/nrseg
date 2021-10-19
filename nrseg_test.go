package nrseg

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNrseg_Run_Default(t *testing.T) {
	dest := t.TempDir()
	tests := [...]struct {
		name string
		want string
		args []string
	}{
		{
			name: "basic",
			want: "./testdata/want",
			args: []string{"nrseg", "-destination", dest, "-i", "ignore", "./testdata/input"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			errs := &bytes.Buffer{}
			if err := Run(tt.args, out, errs, "", ""); err != nil {
				t.Fatalf("run() error = %v", err)
			}
			validate(t, dest, tt.want)
		})
	}
}

func TestNrseg_Run_Inspect(t *testing.T) {
	tests := [...]struct {
		name string
		want string
		args []string
	}{
		{
			name: "basic",
			args: []string{"nrseg", "inspect", "./testdata/input"},
			want: `testdata/input/basic.go:11:1: S.SampleMethod no insert segment
testdata/input/basic.go:16:1: SampleFunc no insert segment
testdata/input/basic.go:21:1: SampleHandler no insert segment
testdata/input/ignore/must_not_change.go:11:1: MustNotChange.SampleMethod no insert segment
testdata/input/ignore/must_not_change.go:16:1: SampleFunc no insert segment
testdata/input/ignore/must_not_change.go:21:1: SampleHandler no insert segment
`,
		},
		{
			name: "ignoreDir",
			args: []string{"nrseg", "inspect", "-i", "ignore", "./testdata/input"},
			want: `testdata/input/basic.go:11:1: S.SampleMethod no insert segment
testdata/input/basic.go:16:1: SampleFunc no insert segment
testdata/input/basic.go:21:1: SampleHandler no insert segment
`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			errs := &bytes.Buffer{}
			if err := Run(tt.args, out, errs, "", ""); !errors.Is(err, ErrFlagTrue) {
				t.Fatalf("want %v, but got %v", ErrFlagTrue, err)
			}
			if out.String() != tt.want {
				t.Errorf("want\n%s\nbut got\n%s", tt.want, out.String())
			}
		})
	}
}

func TestNrseg_run(t *testing.T) {
	type fields struct {
		path      string
		outStream io.Writer
		errStream io.Writer
	}
	tests := [...]struct {
		name   string
		want   string
		fields fields
	}{
		{
			name: "basic",
			want: "./testdata/want",
			fields: fields{
				path:      "./testdata/input",
				outStream: os.Stdout,
				errStream: os.Stderr,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dest := t.TempDir()
			n := &nrseg{
				in:         tt.fields.path,
				dest:       dest,
				ignoreDirs: []string{"testdata", "ignore"},
				outStream:  tt.fields.outStream,
				errStream:  tt.fields.errStream,
			}
			if err := n.run(); err != nil {
				t.Fatalf("run() error = %v", err)
			}
			validate(t, dest, tt.want)
		})
	}
}

func validate(t *testing.T, gotpath, wantpath string) {
	filepath.Walk(gotpath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		rel, err := filepath.Rel(gotpath, path)
		if err != nil {
			t.Errorf("filepath.Rel(): not want error at %q: %v", path, err)
			return err
		}
		wfp := filepath.Join(wantpath, rel)
		wf, err := os.Open(wfp)
		if err != nil {
			t.Errorf("cannot open the wanted file %q: %v", path, err)
			return err
		}
		defer wf.Close()
		want, err := ioutil.ReadAll(wf)
		if err != nil {
			t.Errorf("cannot read the wanted file %q: %v", path, err)
			return err
		}

		gf, err := os.Open(path)
		if err != nil {
			t.Errorf("cannot read the got file %q: %v", path, err)
			return err
		}
		got, err := ioutil.ReadAll(gf)
		if err != nil {
			t.Errorf("cannot read the got file %q: %v", path, err)
			return err
		}
		if diff := cmp.Diff(got, want); len(diff) != 0 {
			t.Errorf("%q -got +want %v", path, diff)
		}

		return nil
	})
}
