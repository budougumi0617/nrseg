package nrseg

import (
	"bytes"
	"errors"
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
			if fd.Body != nil && len(fd.Body.List) > 0 {
				sn := getSegName(fd)
				vn, t := parseParams(f.Imports, fd.Type)
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
	pkg, err := findImport(fs, f)
	if err == nil {
		return pkg, nil
	}
	if errors.Is(err, ErrNoImportNrPkg) {
		astutil.AddImport(fs, f, NewRelicV3Pkg)
		return "", nil
	}

	return "", err
}

var ErrNoImportNrPkg = errors.New("not import newrelic pkg")

func findImport(fs *token.FileSet, f *ast.File) (string, error) {
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
	return "", ErrNoImportNrPkg
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
	arg := &ast.Ident{NamePos: pos, Name: ctxName}
	return skeletonDeferStmt(pos, arg, pkgName, segName)
}

// buildDeferStmt builds the defer statement with *http.Request.
// ex:
//    defer newrelic.FromContext(req.Context()).StartSegment("slow").End()
func buildDeferStmtWithHttpRequest(pos token.Pos, pkgName, reqName, segName string) *ast.DeferStmt {
	arg := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{NamePos: pos, Name: reqName},
			Sel: &ast.Ident{NamePos: pos, Name: "Context"},
		},
		Rparen: pos,
	}
	return skeletonDeferStmt(pos, arg, pkgName, segName)
}

func skeletonDeferStmt(pos token.Pos, fcArg ast.Expr, pkgName, segName string) *ast.DeferStmt {
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
							Args:   []ast.Expr{fcArg},
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

func getSegName(fd *ast.FuncDecl) string {
	var prefix string
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		if rn, ok := fd.Recv.List[0].Type.(*ast.StarExpr); ok {
			if idt, ok := rn.X.(*ast.Ident); ok {
				prefix = toSnake(idt.Name)
			}
		} else if idt, ok := fd.Recv.List[0].Type.(*ast.Ident); ok {
			prefix = toSnake(idt.Name)
		}
	}
	sn := toSnake(fd.Name.Name)
	if len(prefix) != 0 {
		sn = prefix + "_" + sn
	}
	return sn
}

// https://www.golangprograms.com/golang-convert-string-into-snake-case.html
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnake(n string) string {
	snake := matchFirstCap.ReplaceAllString(n, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

const (
	TypeContext     = "context.Context"
	TypeHttpRequest = "*http.Request"
	TypeUnknown     = "Unknown"
)

var types = map[string]string{
	TypeContext:     "\"context\"",
	TypeHttpRequest: "\"net/http\"",
}

func parseParams(is []*ast.ImportSpec, t *ast.FuncType) (string, string) {
	var cname = getImportName(is, TypeContext)
	var hname = getImportName(is, TypeHttpRequest)
	n, typ := "", TypeUnknown
	for _, f := range t.Params.List {
		if se, ok := f.Type.(*ast.SelectorExpr); ok {
			if idt, ok := se.X.(*ast.Ident); ok && idt.Name == cname && se.Sel.Name == "Context" {
				if len(f.Names) > 0 && len(f.Names[0].Name) > 0 {
					return f.Names[0].Name, TypeContext
				}
			}
		}
		if se, ok := f.Type.(*ast.StarExpr); ok {
			if se, ok := se.X.(*ast.SelectorExpr); ok {
				if idt, ok := se.X.(*ast.Ident); ok && idt.Name == hname && se.Sel.Name == "Request" {
					if len(f.Names) > 0 && len(f.Names[0].Name) > 0 {
						n = f.Names[0].Name
						typ = TypeHttpRequest
					}
				}
			}
		}
	}
	return n, typ
}

func getImportName(is []*ast.ImportSpec, typ string) string {
	var def = strings.Replace(strings.Split(typ, ".")[0], "*", "", 1)
	for _, i := range is {
		if i.Name != nil && i.Path != nil && i.Path.Value == types[typ] {
			return i.Name.Name
		}
	}
	return def
}
