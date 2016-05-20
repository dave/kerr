package kerrsource // import "kego.io/kerr/kerrsource"

// ke: {"package": {"complete": true}}

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strconv"

	"bytes"

	"kego.io/kerr"
)

func Process(filename string, in []byte) (out []byte, err error) {

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, in, parser.ParseComments)
	if err != nil {
		return nil, kerr.Wrap("UIXBVGAPWL", err)
	}

	// visitor implements ast.Visitor
	v := &visitor{in, false}
	ast.Walk(v, file)

	// if we made a replacement, re-write the modified file
	if v.Found {
		buf := bytes.NewBuffer(out)
		if err := printer.Fprint(buf, fset, file); err != nil {
			return nil, kerr.Wrap("ETJOALUIWG", err)
		}
		return buf.Bytes(), nil
	}

	return in, nil
}

type visitor struct {
	Bytes []byte
	Found bool
}

func (v *visitor) Visit(node ast.Node) (w ast.Visitor) {

	if node == nil {
		return v
	}

	switch t := node.(type) {
	case *ast.CallExpr:
		name := ""
		pkg := ""
		if f, ok := t.Fun.(*ast.SelectorExpr); ok {
			name = f.Sel.Name
			if i, ok := f.X.(*ast.Ident); ok {
				pkg = i.Name
			}
		}
		if pkg != "kerr" {
			return v
		}
		switch name {
		case "New", "Wrap":
			addId := false
			argsLen := len(t.Args)
			if argsLen == 0 {
				addId = true
			} else if b, ok := t.Args[0].(*ast.BasicLit); ok && b.Kind == token.STRING {
				firstArg, _ := strconv.Unquote(b.Value)
				if !IsId(firstArg) {
					addId = true
				}
			} else {
				addId = true
			}

			if addId {
				id := &ast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote(GetRandomId()),
				}
				t.Args = append([]ast.Expr{id}, t.Args...)
				v.Found = true
			}
		}
	}

	return v
}
