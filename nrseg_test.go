package nrseg

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
			dist := t.TempDir()
			n := &nrseg{
				in:        tt.fields.path,
				dist:      dist,
				outStream: tt.fields.outStream,
				errStream: tt.fields.errStream,
			}
			if err := n.run(); err != nil {
				t.Fatalf("run() error = %v", err)
			}
			validate(t, dist, tt.want)
		})
	}
}

func validate(t *testing.T, gotpath, wantpath string) {
	filepath.Walk(gotpath, func(path string, info os.FileInfo, err error) error {
		// TODO: skip auto generated file
		if filepath.Base(path) == "testdata" {
			return fmt.Errorf("skip testdata dir")
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		rel, err := filepath.Rel(gotpath, path)
		if err != nil {
			t.Errorf("fileapth.Rel(): not want error at %q: %v", path, err)
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
