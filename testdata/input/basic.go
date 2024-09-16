package input

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

// ref: https://github.com/budougumi0617/nrseg/issues/20
func ArgWithouteNameHandler(http.ResponseWriter, *http.Request) {
	fmt.Println("issue #20")
}

// ref: https://github.com/budougumi0617/nrseg/issues/20
func ArgWithouteName(context.Context, string) {
	fmt.Println("issue #20")
}
