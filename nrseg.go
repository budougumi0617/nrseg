package nrseg

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

func Process(filename string, src []byte) ([]byte, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	// import newrelic pkg
	pkg := "newrelic"
	name, err := addImport(fs, f) // importされたpkgの名前
	if err != nil {
		return nil, err
	}
	if len(name) != 0 {
		// change name if named import.
		pkg = name
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			if fd.Body != nil {
				// TODO: no append if exist calling statement of newrelic.FromContext.
				// TODO: get context variable name from function/method argument.
				// TODO: create segment name by function/method name.
				ds := buildDeferStmt(pkg, "ctx", "slow")
				fd.Body.List = append([]ast.Stmt{ds}, fd.Body.List...)
			}
		}
		return true
	})

	// gofmt
	var fmtedBuf bytes.Buffer
	if err := format.Node(&fmtedBuf, fs, f); err != nil {
		return nil, err
	}

	// goimports
	igot, err := imports.Process(filename, fmtedBuf.Bytes(), nil)
	if err != nil {
		return nil, err
	}

	return igot, nil
}

const NewRelicV3Pkg = "github.com/newrelic/go-agent/v3/newrelic"

func addImport(fs *token.FileSet, f *ast.File) (string, error) {
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return "", err
		}
		if path == NewRelicV3Pkg {
			// import already.
			return spec.Name.Name, nil
		}
	}
	astutil.AddImport(fs, f, NewRelicV3Pkg)
	return "", nil
}

// buildDeferStmt builds the defer statement with args.
// ex:
//    defer newrelic.FromContext(ctx).StartSegment("slow").End()
func buildDeferStmt(pkgName, ctxName, segName string) *ast.DeferStmt {
	return &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   &ast.Ident{Name: pkgName},
								Sel: &ast.Ident{Name: "FromContext"},
							},
							Args: []ast.Expr{&ast.Ident{Name: ctxName}},
						},
						Sel: &ast.Ident{Name: "StartSegment"},
					},
					Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(segName)}},
				},
				Sel: &ast.Ident{Name: "End"},
			},
		},
	}
}
