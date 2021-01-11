package input

import (
	"fmt"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// no insert because ignore comment.
// nrseg:ignore this is test.
func IgnoreHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Hello, %q", req.URL.Path)
}

// no insert because called already.
func AlreadyHandler(w http.ResponseWriter, req *http.Request) {
	defer newrelic.FromContext(req.Context()).StartSegment("already_handler").End()
	fmt.Fprintf(w, "Hello, %q", req.URL.Path)
}
