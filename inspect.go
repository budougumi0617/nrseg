package nrseg

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
)

func (nrseg *nrseg) Inspect(filename string, src []byte) error {
	if len(src) != 0 && c.Match(src) {
		return nil
	}
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, filename, src, parser.ParseComments)
	if err != nil {
		return err
	}
	// import newrelic pkg
	pkg := "newrelic"
	name, err := findImport(fs, f) // importされたpkgの名前
	if err != nil && !errors.Is(err, ErrNoImportNrPkg) {
		return err
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
				if _, t := parseParams(f.Imports, fd.Type); !(t == TypeContext || t == TypeHttpRequest) {
					return false
				}
				if !existFromContext(pkg, fd.Body.List[0]) {
					nrseg.errFlag = true
					nrseg.reportf(filename, fs, fd.Pos(), fd)
				}
				return false
			}
		}
		return true
	})

	return nil
}
