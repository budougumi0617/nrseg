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
`,
			want: `package main

import (
	"context"
	"fmt"

	"github.com/newrelic/go-agent/v3/newrelic"
)

type S struct{}

func (s *S) SampleMethod(ctx context.Context) {
	defer newrelic.FromContext(ctx).StartSegment("slow").End()
	fmt.Println("Hello, playground")
	fmt.Println("end function")
}

func SampleFunc(ctx context.Context) {
	defer newrelic.FromContext(ctx).StartSegment("slow").End()
	fmt.Println("Hello, playground")
	fmt.Println("end function")
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
