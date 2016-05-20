package kerrsource_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"strconv"

	"golang.org/x/net/context"
	"kego.io/json"
	"kego.io/kerr"
	"kego.io/kerr/assert"
	"kego.io/kerr/kerrsource"
	"kego.io/process/packages"
)

var pkgs = map[string]*pkgDef{}
var all = map[string]*errDef{}

func TestAll(t *testing.T) {
	dir, err := packages.GetDirFromPackage(context.Background(), "kego.io")
	assert.NoError(t, err)

	// walk each file in the working directory
	walker := func(path string, file os.FileInfo, err error) error {
		// skip anything starting with a period, or not ending in .go.
		if strings.HasPrefix(path, ".") || !strings.HasSuffix(path, ".go") {
			return nil
		}
		return walkFile(path, t)
	}
	filepath.Walk(dir, walker)

	untested := map[string][]string{}
	for id, def := range all {
		if !def.thrown && (def.tested || def.skipped) {
			// Well, if the test isn't thrown the test should fail. Let's test for it here anyway.
			assert.Fail(t, "Error tested but not thrown", id)
		}
		if def.thrown && def.new && !def.tested && !def.skipped {
			arr, ok := untested[def.pkg]
			if !ok {
				arr = []string{}
			}
			untested[def.pkg] = append(arr, id)
		}
	}

	//for pkg, def := range pkgs {
	//	if !def.tested && !def.notest {
	//assert.Fail(t, fmt.Sprintf("%s has no tests.", pkg))
	//	}
	//}

	//disabled
	return

	if len(untested) > 0 {
		for pkg, tests := range untested {
			assert.Fail(t, fmt.Sprintf("Errors thrown in %s but not tested: %v", pkg, tests))
		}
	}
}

func walkFile(path string, t *testing.T) error {

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return kerr.Wrap("GJULIOKHIR", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return kerr.Wrap("HMRIYHYYYH", err)
	}

	pkg, err := packages.GetPackageFromDir(context.Background(), filepath.Dir(path))
	if err != nil {
		return kerr.Wrap("FIPPWAKAGK", err)
	}

	// ignore anything that starts with kego.io/demo/
	if strings.HasPrefix(pkg, "kego.io/demo/") {
		return nil
	}

	def := getPkgDef(pkg)
	if strings.HasSuffix(path, "_test.go") {
		def.tested = true
	}

	kerrName := ""
	assertName := ""
	for _, is := range file.Imports {
		importPath, _ := strconv.Unquote(is.Path.Value)
		switch importPath {
		case "kego.io/kerr":
			if is.Name != nil {
				kerrName, _ = strconv.Unquote(is.Name.Name)
			} else {
				kerrName = "kerr"
			}
		case "kego.io/kerr/assert":
			if is.Name != nil {
				assertName, _ = strconv.Unquote(is.Name.Name)
			} else {
				assertName = "assert"
			}
		}
	}

	// visitor implements ast.Visitor
	v := &visitor{
		Bytes:      b,
		t:          t,
		kerrName:   kerrName,
		assertName: assertName,
		file:       path,
		pkg:        pkg,
	}
	ast.Walk(v, file)

	return nil
}

type visitor struct {
	Bytes      []byte
	t          *testing.T
	kerrName   string
	assertName string
	file       string
	pkg        string
}

type pkgDef struct {
	tested bool
	notest bool
}
type errDef struct {
	id      string
	new     bool
	thrown  bool
	tested  bool
	skipped bool
	pkg     string
}

func (v *visitor) Visit(node ast.Node) (w ast.Visitor) {

	if node == nil {
		return v
	}

	switch ty := node.(type) {
	case *ast.File:
		for _, cg := range ty.Comments {
			for _, c := range cg.List {
				if strings.HasPrefix(c.Text, "// ke: ") {
					val := struct{ Package struct{ Notest bool } }{}
					err := json.UnmarshalPlain([]byte(c.Text[7:]), &val)
					assert.NoError(v.t, err)
					if val.Package.Notest {
						def := getPkgDef(v.pkg)
						def.notest = true
					}
				}
			}
		}
	case *ast.CallExpr:
		name := ""
		pkg := ""
		if f, ok := ty.Fun.(*ast.SelectorExpr); ok {
			name = f.Sel.Name
			if i, ok := f.X.(*ast.Ident); ok {
				pkg = i.Name
			}
			if pkg == "" {
				return v
			}
			if pkg == v.kerrName && (name == "New" || name == "Wrap") {
				def := getErrData(v.t, ty.Args, 0, v.file)
				if def.thrown {
					assert.Fail(v.t, fmt.Sprint("Duplicate kerr id: ", def.id))
				}
				def.pkg = v.pkg
				def.new = name == "New"
				def.thrown = true
			} else if pkg == v.assertName && (name == "IsError" || name == "HasError") {
				def := getErrData(v.t, ty.Args, 2, v.file)
				def.tested = true
			} else if pkg == v.assertName && (name == "SkipError") {
				def := getErrData(v.t, ty.Args, 0, v.file)
				def.skipped = true
			}

		}
	}
	return v
}

func getErrData(t *testing.T, args []ast.Expr, arg int, file string) *errDef {
	assert.True(t, len(args) > arg, "Not enough args (%s)", file)
	b, ok := args[arg].(*ast.BasicLit)
	assert.True(t, ok, "Arg should be *ast.BasicLit (%s)", file)
	assert.Equal(t, b.Kind, token.STRING, "kind should be token.STRING (%s)", file)
	id, err := strconv.Unquote(b.Value)
	assert.NoError(t, err, "Error unquoting arg (%s)", file)
	assert.True(t, kerrsource.IsId(id), "Invalid kerr ID %s (%s)", id, file)
	def, ok := all[id]
	if ok {
		return def
	}
	def = &errDef{}
	def.id = id
	all[id] = def
	return def
}

func getPkgDef(path string) *pkgDef {
	def, ok := pkgs[path]
	if ok {
		return def
	}
	def = &pkgDef{}
	pkgs[path] = def
	return def
}
