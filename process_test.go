package nrseg

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProcess(t *testing.T) {
	t.Parallel()
	tests := [...]struct {
		name, src, want string
	}{
		{
			name: "basicProcess",
			src: `package main

import (
	"context"
	"fmt"
	"net/http"
)

type S struct{}

func (s *S) SampleMethod(ctx context.Context) {
	fmt.Println("Hello, playground")
	fmt.Println("end function")
}

func SampleFunc(ctx context.Context) {
	fmt.Println("Hello, playground")
	fmt.Println("end function")
}

func SampleHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Hello, %q", req.URL.Path)
}
`,
			want: `package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
)

type S struct{}

func (s *S) SampleMethod(ctx context.Context) {
	defer newrelic.FromContext(ctx).StartSegment("sample_method").End()
	fmt.Println("Hello, playground")
	fmt.Println("end function")
}

func SampleFunc(ctx context.Context) {
	defer newrelic.FromContext(ctx).StartSegment("sample_func").End()
	fmt.Println("Hello, playground")
	fmt.Println("end function")
}

func SampleHandler(w http.ResponseWriter, req *http.Request) {
	defer newrelic.FromContext(req.Context()).StartSegment("sample_handler").End()
	fmt.Fprintf(w, "Hello, %q", req.URL.Path)
}
`,
		},
		{
			name: "UseApplication",
			src: `package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/integrations/nrgorilla"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func NewRouter(
	nrapp *newrelic.Application,
) http.Handler {
	router := mux.NewRouter()
	router.Use(nrgorilla.Middleware(nrapp))

	return router
}

func SampleHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Hello, %q", req.URL.Path)
}
`,
			want: `package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/integrations/nrgorilla"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func NewRouter(
	nrapp *newrelic.Application,
) http.Handler {
	router := mux.NewRouter()
	router.Use(nrgorilla.Middleware(nrapp))

	return router
}

func SampleHandler(w http.ResponseWriter, req *http.Request) {
	defer newrelic.FromContext(req.Context()).StartSegment("sample_handler").End()
	fmt.Fprintf(w, "Hello, %q", req.URL.Path)
}
`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Process("", []byte(tt.src))
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(got, []byte(tt.want)); len(diff) != 0 {
				t.Errorf("-got +want %v", diff)
			}
		})
	}
}

func Test_genSegName(t *testing.T) {
	t.Parallel()
	tests := [...]struct {
		name    string
		n, want string
	}{
		{name: "Simple", n: "Simple", want: "simple"},
		{name: "Camel", n: "camelCase", want: "camel_case"},
		{name: "Pascal", n: "PascalCase", want: "pascal_case"},
		{name: "HTML", n: "SaveHTML", want: "save_html"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := genSegName(tt.n); got != tt.want {
				t.Errorf("genSegName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_parseParams(t *testing.T) {
	t.Parallel()
	tests := [...]struct {
		name     string
		src      string
		wantName string
		wantType string
	}{
		{
			name: "Context",
			src: `
package main

import (
	"context"
)

func Hoge(ctx context.Context) {}
`,
			wantName: "ctx", wantType: TypeContext,
		},
		{
			name: "Http",
			src: `
package main

import (
	"net/http"
)

func SampleHandler(w http.ResponseWriter, req *http.Request) {}
`,
			wantName: "req", wantType: TypeHttpRequest,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := token.NewFileSet()
			f, err := parser.ParseFile(fs, "sample.go", tt.src, parser.Mode(0))
			if err != nil {
				t.Fatal(err)
			}
			var decl *ast.FuncDecl
			for _, d := range f.Decls {
				if fd, ok := d.(*ast.FuncDecl); ok {
					decl = fd
					break
				}
			}
			name, gtype := parseParams(decl.Type)
			if name != tt.wantName {
				t.Errorf("parseParams() name = %q, want %q", name, tt.wantName)
			}
			if gtype != tt.wantType {
				t.Errorf("parseParams() type = %q, want %q", gtype, tt.wantType)
			}
		})
	}
}
