package scanner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"encoding/json"

	"github.com/davelondon/gopackages"
	"github.com/davelondon/kerr"
	"github.com/davelondon/kerr/ksrc"
)

// ke: {"package": {"notest": true}}

var baseDir string

type Source struct {
	Wraps            []ErrDef
	Panics           []PosDef
	ExcludedBlocks   []PosDef
	ExcludedPackages map[string]bool
	ExcludedFiles    map[string]bool
	ExcludedFuncs    []FuncDef
	CompletePackages map[string]bool
	Skipped          map[string]bool
	All              map[string]ErrDef
	JsTestPackages   map[string]bool
}
type ErrDef struct {
	Id   string
	File string
	Line int
}
type PosDef struct {
	File string
	Line int
}
type FuncDef struct {
	File      string
	LineStart int
	LineEnd   int
}

var source = &Source{
	Skipped:          map[string]bool{},
	ExcludedPackages: map[string]bool{},
	ExcludedFiles:    map[string]bool{},
	CompletePackages: map[string]bool{},
	All:              map[string]ErrDef{},
	JsTestPackages:   map[string]bool{},
}

func Get(dir string) (*Source, error) {
	baseDir = dir

	// walk each file in the working directory
	walker := func(filename string, file os.FileInfo, err error) error {
		if strings.HasPrefix(getParentDir(filename), ".") {
			return filepath.SkipDir
		}
		if strings.HasPrefix(file.Name(), ".") {
			return nil
		}
		if file.IsDir() {
			return nil
		}
		if strings.HasSuffix(filename, ".go") {
			return scanFile(filename)
		}
		return nil
	}
	if err := filepath.Walk(baseDir, walker); err != nil {
		return nil, kerr.Wrap("UIBJTFCKHQ", err)
	}

	return source, nil
}

func getParentDir(filename string) string {
	dir, _ := filepath.Split(filename)
	dirs := strings.Split(dir, string(os.PathSeparator))
	return dirs[len(dirs)-2]
}

func scanFile(filename string) error {

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return kerr.Wrap("DNMNGWPEYO", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return kerr.Wrap("WTIOTIMQDV", err)
	}

	gopathdir := gopackages.GetCurrentGopath(os.Getenv("GOPATH"), filepath.Dir(filename))

	pkg, err := gopackages.GetPackageFromDir(os.Getenv("GOPATH"), filepath.Dir(filename))
	if err != nil {
		return kerr.Wrap("PJFJLKNVBO", err)
	}

	kerrName := ""
	assertName := ""
	for _, is := range file.Imports {
		importPath, _ := strconv.Unquote(is.Path.Value)
		switch importPath {
		case "github.com/davelondon/kerr":
			if is.Name != nil {
				kerrName, _ = strconv.Unquote(is.Name.Name)
			} else {
				kerrName = "kerr"
			}
		case "github.com/davelondon/ktest/assert":
			if is.Name != nil {
				assertName, _ = strconv.Unquote(is.Name.Name)
			} else {
				assertName = "assert"
			}
		}
	}

	relfilename, err := filepath.Rel(filepath.Join(gopathdir, "src"), filename)
	if err != nil {
		return kerr.Wrap("WSDPGDGMDH", err)
	}

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "// ke: ") {
				val := struct {
					Block struct {
						Notest bool
					}
					Package struct {
						Jstest   bool
						Notest   bool
						Complete bool
					}
					File struct {
						Notest bool
					}
				}{}
				err := json.Unmarshal([]byte(c.Text[7:]), &val)
				if err != nil {
					return kerr.Wrap("LAWVOJLLRJ", err)
				}
				if val.Block.Notest {
					source.ExcludedBlocks = append(source.ExcludedBlocks, PosDef{
						File: relfilename,
						Line: fset.Position(c.Pos()).Line,
					})
				}
				if val.Package.Complete {
					source.CompletePackages[pkg] = true
				}
				if val.Package.Notest {
					source.ExcludedPackages[pkg] = true
				}
				if val.Package.Jstest {
					source.JsTestPackages[pkg] = true
				}
				if val.File.Notest {
					source.ExcludedFiles[relfilename] = true
				}
			}
		}
	}

	// visitor implements ast.Visitor
	v := &visitor{
		Bytes:      b,
		kerrName:   kerrName,
		assertName: assertName,
		file:       relfilename,
		pkg:        pkg,
		fset:       fset,
	}
	ast.Walk(v, file)

	return v.err
}

type visitor struct {
	Bytes      []byte
	t          *testing.T
	kerrName   string
	assertName string
	file       string
	pkg        string
	fset       *token.FileSet
	err        error
}

func (v *visitor) Visit(node ast.Node) (w ast.Visitor) {

	if node == nil {
		return v
	}

	switch ob := node.(type) {
	/*
		case *ast.File:

			for _, cg := range ty.Comments {
				for _, c := range cg.List {
						if strings.HasPrefix(c.Text, "// ke: ") {
							val := struct{ Notest bool }{}
							err := json.UnmarshalPlain([]byte(c.Text[7:]), &val)
							require.NoError(v.t, err)
							if val.Notest {
								def := getPkgDef(v.pkg)
								def.notest = true
							}
						}
				}
			}
	*/
	case *ast.FuncDecl:
		if ob.Doc == nil {
			return v
		}
		for _, c := range ob.Doc.List {
			if strings.HasPrefix(c.Text, "// ke: ") {
				val := struct {
					Func struct {
						Notest bool
					}
				}{}
				err := json.Unmarshal([]byte(c.Text[7:]), &val)
				if err != nil {
					v.err = err
					return nil
				}
				if val.Func.Notest {
					source.ExcludedFuncs = append(source.ExcludedFuncs, FuncDef{
						File:      v.file,
						LineStart: v.fset.Position(ob.Pos()).Line,
						LineEnd:   v.fset.Position(ob.End()).Line,
					})
				}
			}
		}
	case *ast.CallExpr:
		name := ""
		pkg := ""
		if i, ok := ob.Fun.(*ast.Ident); ok {
			if i.Name == "panic" {
				source.Panics = append(source.Panics, PosDef{
					File: v.file,
					Line: v.fset.Position(ob.Pos()).Line,
				})
			}
		}
		if f, ok := ob.Fun.(*ast.SelectorExpr); ok {
			name = f.Sel.Name
			if i, ok := f.X.(*ast.Ident); ok {
				pkg = i.Name
			}

			if pkg == "" {
				return v
			}
			if pkg == v.kerrName && name == "Wrap" {
				id, err := getErrorId(v.t, ob.Args, 0, v.file)
				if err != nil {
					v.err = err
					return nil
				}
				source.Wraps = append(source.Wraps, ErrDef{
					Id:   id,
					File: v.file,
					Line: v.fset.Position(ob.Pos()).Line,
				})
			} else if pkg == v.kerrName && name == "New" {
				id, err := getErrorId(v.t, ob.Args, 0, v.file)
				if err != nil {
					v.err = err
					return nil
				}
				source.All[id] = ErrDef{
					Id:   id,
					File: v.file,
					Line: v.fset.Position(ob.Pos()).Line,
				}
			} else if pkg == v.assertName && (name == "SkipError") {
				id, err := getErrorId(v.t, ob.Args, 0, v.file)
				if err != nil {
					v.err = err
					return nil
				}
				source.Skipped[id] = true
			} /*else if pkg == v.assertName && (name == "IsError" || name == "HasError") {
				def := getErrData(v.t, ty.Args, 2, v.file)
				def.tested = true
			} else if pkg == v.assertName && (name == "SkipError") {
				def := getErrData(v.t, ty.Args, 0, v.file)
				def.skipped = true
			}*/

		}
	}
	return v
}

func getErrorId(t *testing.T, args []ast.Expr, arg int, file string) (string, error) {
	if len(args) <= arg {
		return "", kerr.New("FLKWAYFFYL", "Not enough args (%s)", file)
	}
	b, ok := args[arg].(*ast.BasicLit)
	if !ok {
		return "", kerr.New("YFKEYRIUWJ", "Arg should be *ast.BasicLit (%s)", file)
	}
	if b.Kind != token.STRING {
		return "", kerr.New("YWPXGGRLEL", "kind should be token.STRING (%s)", file)
	}
	id, err := strconv.Unquote(b.Value)
	if err != nil {
		return "", kerr.Wrap("NTUDBTYOQY", err)
	}
	if !ksrc.IsId(id) {
		return "", kerr.New("YLTSLGODKR", "Invalid kerr ID %s (%s)", id, file)
	}
	return id, nil
}
