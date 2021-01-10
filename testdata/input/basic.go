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
