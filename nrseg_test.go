package nrseg

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProcess(t *testing.T) {
	tests := []struct {
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := Process("", []byte(tt.src))
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(got, []byte(tt.want)); len(diff) != 0 {
				// t.Errorf("want\n%s\ngot\n%s\n", fwant, got)
				t.Errorf("-got +want %v", diff)
			}
		})
	}
}

func Test_genSegName(t *testing.T) {
	tests := []struct {
		name    string
		n, want string
	}{
		{name: "Simple", n: "Simple", want: "simple"},
		{name: "Camel", n: "camelCase", want: "camel_case"},
		{name: "Pascal", n: "PascalCase", want: "pascal_case"},
		{name: "HTML", n: "SaveHTML", want: "save_html"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := genSegName(tt.n); got != tt.want {
				t.Errorf("genSegName() = %q, want %q", got, tt.want)
			}
		})
	}
}
