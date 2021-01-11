package nrseg

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

func Process(filename string, src []byte) ([]byte, error) {
	if len(src) != 0 && c.Match(src) {
		return src, nil
	}
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
			if findIgnoreComment(fd.Doc) {
				return false
			}
			if fd.Body != nil {
				sn := genSegName(fd.Name.Name)
				vn, t := parseParams(fd.Type)
				var ds ast.Stmt
				switch t {
				case TypeContext:
					ds = buildDeferStmt(fd.Body.Lbrace, pkg, vn, sn)
				case TypeHttpRequest:
					ds = buildDeferStmtWithHttpRequest(fd.Body.Lbrace, pkg, vn, sn)
				case TypeUnknown:
					return false
				}

				if !existFromContext(pkg, fd.Body.List[0]) {
					fd.Body.List = append([]ast.Stmt{ds}, fd.Body.List...)
				}
				return false
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
		// import already.
		if path == NewRelicV3Pkg {
			if spec.Name != nil {
				return spec.Name.Name, nil
			}
			return "", nil
		}
	}
	astutil.AddImport(fs, f, NewRelicV3Pkg)
	return "", nil
}

var nrignoreReg = regexp.MustCompile("(?m)^// nrseg:ignore .*$")

func findIgnoreComment(cg *ast.CommentGroup) bool {
	if cg == nil {
		return false
	}
	for _, c := range cg.List {
		if nrignoreReg.MatchString(c.Text) {
			return true
		}
	}
	return false
}

func existFromContext(pn string, s ast.Stmt) bool {
	var result bool
	ast.Inspect(s, func(n ast.Node) bool {
		if se, ok := n.(*ast.SelectorExpr); ok {
			if idt, ok := se.X.(*ast.Ident); ok && idt.Name == pn && se.Sel.Name == "FromContext" {
				result = true
				return false
			}
		}
		return true
	})
	return result
}

// buildDeferStmt builds the defer statement with args.
// ex:
//    defer newrelic.FromContext(ctx).StartSegment("slow").End()
func buildDeferStmt(pos token.Pos, pkgName, ctxName, segName string) *ast.DeferStmt {
	return &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   &ast.Ident{NamePos: pos, Name: pkgName},
								Sel: &ast.Ident{NamePos: pos, Name: "FromContext"},
							},
							Args: []ast.Expr{&ast.Ident{NamePos: pos, Name: ctxName}},
						},
						Sel: &ast.Ident{NamePos: pos, Name: "StartSegment"},
					},
					Args: []ast.Expr{&ast.BasicLit{
						ValuePos: pos,
						Kind:     token.STRING,
						Value:    strconv.Quote(segName),
					}},
				},
				Sel: &ast.Ident{NamePos: pos, Name: "End"},
			},
			Rparen: pos,
		},
	}
}

// buildDeferStmt builds the defer statement with *http.Request.
// ex:
//    defer newrelic.FromContext(req.Context()).StartSegment("slow").End()
func buildDeferStmtWithHttpRequest(pos token.Pos, pkgName, reqName, segName string) *ast.DeferStmt {
	return &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   &ast.Ident{NamePos: pos, Name: pkgName},
								Sel: &ast.Ident{NamePos: pos, Name: "FromContext"},
							},
							Lparen: pos,
							Args: []ast.Expr{
								&ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X:   &ast.Ident{NamePos: pos, Name: reqName},
										Sel: &ast.Ident{NamePos: pos, Name: "Context"},
									},
									Rparen: pos,
								},
							},
							Rparen: pos,
						},
						Sel: &ast.Ident{NamePos: pos, Name: "StartSegment"},
					},
					Args: []ast.Expr{&ast.BasicLit{
						ValuePos: pos,
						Kind:     token.STRING,
						Value:    strconv.Quote(segName),
					}},
				},
				Sel: &ast.Ident{NamePos: pos, Name: "End"},
			},
			Rparen: pos,
		},
	}
}

// https://www.golangprograms.com/golang-convert-string-into-snake-case.html
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func genSegName(n string) string {
	snake := matchFirstCap.ReplaceAllString(n, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

const (
	TypeContext     = "context.Context"
	TypeHttpRequest = "*http.Request"
	TypeUnknown     = "Unknown"
)

func parseParams(t *ast.FuncType) (string, string) {
	n, typ := "", TypeUnknown
	for _, f := range t.Params.List {
		if se, ok := f.Type.(*ast.SelectorExpr); ok {
			// TODO: support named import
			if idt, ok := se.X.(*ast.Ident); ok && idt.Name == "context" && se.Sel.Name == "Context" {
				return f.Names[0].Name, TypeContext
			}
		}
		if se, ok := f.Type.(*ast.StarExpr); ok {
			if se, ok := se.X.(*ast.SelectorExpr); ok {
				// TODO: support named import
				if idt, ok := se.X.(*ast.Ident); ok && idt.Name == "http" && se.Sel.Name == "Request" {
					n = f.Names[0].Name
					typ = TypeHttpRequest
				}
			}
		}
	}
	return n, typ
}
