nrseg
===
[![Go Reference](https://pkg.go.dev/badge/github.com/budougumi0617/nrseg.svg)](https://pkg.go.dev/github.com/budougumi0617/nrseg)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![test](https://github.com/budougumi0617/nrseg/workflows/test/badge.svg)](https://github.com/budougumi0617/nrseg/actions?query=workflow%3Atest)
[![reviewdog](https://github.com/budougumi0617/nrseg/workflows/reviewdog/badge.svg)](https://github.com/budougumi0617/nrseg/actions?query=workflow%3Areviewdog)

## Background
https://docs.newrelic.com/docs/agents/go-agent/instrumentation/instrument-go-segments

NewRelic is excellent o11y service, but if we use Newrelic in Go app, we need to insert `segment` into every function/method to measure the time taken by functions and code blocks.
For example, we can use with `newrelic.FromContext` and `defer` statement.

```go
func SampleFunc(ctx context.Context) {
  defer newrelic.FromContext(ctx).StartSegment("sample_func").End()
  // do anything...
}
```

If there is your application in production already, you must add a segment into any function/method. It is a very time-consuming and tedious task.

## Description
`nrseg` is cli tool for insert segment into all function/method in specified directory.

Before code is below,
```go
package input

import (
  "context"
  "fmt"
  "net/http"
)

type FooBar struct{}

func (f *FooBar) SampleMethod(ctx context.Context) {
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

// nrseg:ignore you can be ignored if you want to not insert segment.
func IgnoreHandler(w http.ResponseWriter, req *http.Request) {
  fmt.Fprintf(w, "Hello, %q", req.URL.Path)
}
```

After execute `nrseg`, modified code is below. 

```go
package input

import (
  "context"
  "fmt"
  "net/http"

  "github.com/newrelic/go-agent/v3/newrelic"
)

type FooBar struct{}

func (f *FooBar) SampleMethod(ctx context.Context) {
  defer newrelic.FromContext(ctx).StartSegment("foo_bar_sample_method").End()
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

// nrseg:ignore you can be ignored if you want to not insert segment.
func IgnoreHandler(w http.ResponseWriter, req *http.Request) {
  fmt.Fprintf(w, "Hello, %q", req.URL.Path)
}
```

### Features
- [x] Insert `Function segments` into the function with following arguments.
  - The function/method signature has `context.Context`.
    - `defer newrelic.FromContext(ctx).StartSegment("func_name").End()`
  - The function/method signature has `*http.Request`.
      - `defer newrelic.FromContext(req.Context()).StartSegment("func_name").End()`
- [x] Support any variable name of `context.Context`/`*http.Request`.
- [x] Use function/method name to segment name.
- [x] This processing is recursively repeated.
- [x] Able to ignore function/method by `nrseg:ignore` comment.
- [x] Ignore specified directories with cli option `-i`/`-ignore`.
- [ ] Remove all `Function segments`
- [ ] Add: `dry-run` option
- [ ] Validate: Show a function that doesn't call the segment.
- [ ] Support anonymous function

## Synopsis
```
$ nrseg -i testuitl ./
```

## Options

```
$ nrseg -h
Insert function segments into any function/method for Newrelic APM.

Usage of nrseg:
  -i string
        ignore directory names. ex: foo,bar,baz
        (testdata directory is always ignored.)
  -ignore string
        ignore directory names. ex: foo,bar,baz
        (testdata directory is always ignored.)
  -v    print version information and quit.
  -version
        print version information and quit.
exit status 1

```

## Limitation
nrseg inserts only `function segments`, so we need the initialize of Newrelic manually. 

- [Install New Relic for Go][segment]
- [Monitor a transaction by wrapping an HTTP handler][nr_handler]

[nr_handler]: https://docs.newrelic.com/docs/agents/go-agent/instrumentation/instrument-go-transactions#http-handler-txns
[segment]: https://docs.newrelic.com/docs/agents/go-agent/installation/install-new-relic-go

If we want to adopt Newrelic to our application, , we write initialize, and newrelic.WrapHandleFunc manually before execute this tool.
```go
app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("my_application"),
		newrelic.ConfigLicense(newrelicLicenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
```

## Installation

```
$ go install github.com/budougumi0617/nrseg/cmd/nrseg
```

Built binaries are available on gihub releases. https://github.com/budougumi0617/nrseg/releases

### MacOS
If you want to install on MacOS, you can use Homebrew.
```
brew install budougumi0617/tap/nrseg
```

## Contribution
1. Fork ([https://github.com/budougumi0617/nrseg/fork](https://github.com/budougumi0617/nrseg/fork))
2. Create a feature branch
3. Commit your changes
4. Rebase your local changes against the master branch
5. Run test suite with the `go test ./...` command and confirm that it passes
6. Run `gofmt -s`
7. Create new Pull Request

## License

[MIT](https://github.com/budougumi0617/nrseg/blob/master/LICENSE)

## Author
[budougumi0617](https://github.com/budougumi0617)

