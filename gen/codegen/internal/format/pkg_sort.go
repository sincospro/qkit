package format

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io/ioutil"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func SortImports(fset *token.FileSet, f *ast.File, file string) error {
	ast.SortImports(fset, f)
	dir := path.Dir(file)

	for _, decl := range f.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.IMPORT || len(d.Specs) == 0 {
			break
		}
		g := &GroupSet{}
		for i := range d.Specs {
			s := d.Specs[i].(*ast.ImportSpec)
			path, _ := strconv.Unquote(s.Path.Value)

			if Stds[path] {
				g.AppendStd(path, s)
				continue
			}
			if strings.Contains(dir, path) {
				g.AppendLocal(path, s)
				continue
			}
			g.AppendVendor(path, s)
		}
		_fset, _f, err := Parse(file, bytes.Replace(
			FormatNode(fset, f),
			FormatNode(fset, d),
			g.Bytes(),
			1,
		))
		if err != nil {
			fmt.Println(".....", err)
			return err
		}
		*fset, *f = *_fset, *_f
	}

	return nil
}

func FormatNode(fset *token.FileSet, node ast.Node) []byte {
	buf := bytes.NewBuffer(nil)
	if err := format.Node(buf, fset, node); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

type dep struct {
	Pkg  string
	Spec *ast.ImportSpec
}

type GroupSet [4][]*dep

func (g *GroupSet) Bytes() []byte {
	buf := bytes.NewBuffer(nil)

	buf.WriteString("import ")
	buf.WriteRune('(')
	for _, deps := range *g {
		for _, d := range deps {
			buf.WriteRune('\n')
			if d.Spec.Doc != nil {
				for _, c := range d.Spec.Doc.List {
					buf.WriteString(c.Text)
					buf.WriteRune('\n')
				}
			}
			if d.Spec.Name != nil {
				buf.WriteString(d.Spec.Name.String())
				buf.WriteRune(' ')
			}
			buf.WriteString(d.Spec.Path.Value)
			if d.Spec.Comment != nil {
				for _, c := range d.Spec.Comment.List {
					buf.WriteString(c.Text)
				}
			}
		}
		buf.WriteRune('\n')
	}
	buf.WriteRune(')')

	return buf.Bytes()
}

func (g *GroupSet) append(idx int, pkg string, spec *ast.ImportSpec) {
	(*g)[idx] = append((*g)[idx], &dep{Pkg: pkg, Spec: spec})
}

func (g *GroupSet) AppendStd(pkg string, spec *ast.ImportSpec)    { g.append(0, pkg, spec) }
func (g *GroupSet) AppendVendor(pkg string, spec *ast.ImportSpec) { g.append(1, pkg, spec) }
func (g *GroupSet) AppendLocal(pkg string, spec *ast.ImportSpec)  { g.append(2, pkg, spec) }

type StdLibSet map[string]bool

func (s StdLibSet) WalkInit(root, prefix string) {
	ds, _ := ioutil.ReadDir(root)
	for _, d := range ds {
		if !d.IsDir() {
			continue
		}
		pkg := d.Name()
		if prefix != "" {
			pkg = filepath.Join(prefix, pkg)
		}
		s.WalkInit(filepath.Join(root, pkg), pkg)
		s[pkg] = true
	}
}

var Stds StdLibSet

func init() {
	Stds = make(StdLibSet)
	Stds.WalkInit(path.Join(runtime.GOROOT(), "src"), "")
}
