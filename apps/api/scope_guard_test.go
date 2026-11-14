package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

var (
	terminal = map[string]bool{"First": true, "Take": true, "Find": true, "Last": true, "Scan": true, "Count": true,
		"Pluck": true, "Delete": true, "Update": true, "Updates": true, "FirstOrCreate": true}
	rawTableAllowed = []string{"internal/retention/", "internal/documentation/", "schemas/log_entry.go"}
)

type scopeGuard struct {
	fset     *token.FileSet
	scoping  map[string]bool
	findings []string
}

func isLogEntryType(expr ast.Expr) bool {
	if array, ok := expr.(*ast.ArrayType); ok {
		expr = array.Elt
	}
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "LogEntry" && isIdent(sel.X, "schemas")
}

func isIdent(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func (g *scopeGuard) report(path string, pos token.Pos, what string) {
	g.findings = append(g.findings, g.fset.Position(pos).String()+": "+what)
}

// scopedChain reports whether expr, a chain of calls, passes through
// logfilter or through a helper that does, following a variable back to its
// first assignment in fn.
func (g *scopeGuard) scopedChain(expr ast.Expr, fn *ast.FuncDecl, depth int) bool {
	for depth < 8 {
		switch node := expr.(type) {
		case *ast.CallExpr:
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if isIdent(sel.X, "logfilter") || g.scoping[sel.Sel.Name] {
					return true
				}
				expr = sel.X
				continue
			}
			ident, ok := node.Fun.(*ast.Ident)
			return ok && g.scoping[ident.Name]
		case *ast.SelectorExpr:
			expr = node.X
		case *ast.Ident:
			rhs := firstAssignment(fn, node.Name)
			if rhs == nil {
				return false
			}
			expr, depth = rhs, depth+1
		default:
			return false
		}
	}
	return false
}

func firstAssignment(fn *ast.FuncDecl, name string) ast.Expr {
	var found ast.Expr
	ast.Inspect(fn, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || found != nil || len(assign.Lhs) != len(assign.Rhs) {
			return found == nil
		}
		for i, lhs := range assign.Lhs {
			if isIdent(lhs, name) && assign.Tok == token.DEFINE {
				found = assign.Rhs[i]
			}
		}
		return found == nil
	})
	return found
}

func logEntryVars(fn *ast.FuncDecl) map[string]bool {
	vars := map[string]bool{}
	ast.Inspect(fn, func(n ast.Node) bool {
		if spec, ok := n.(*ast.ValueSpec); ok && spec.Type != nil && isLogEntryType(spec.Type) {
			for _, name := range spec.Names {
				vars[name.Name] = true
			}
		}
		return true
	})
	return vars
}

func argNames(args []ast.Expr) []string {
	var out []string
	for _, arg := range args {
		if unary, ok := arg.(*ast.UnaryExpr); ok && unary.Op == token.AND {
			arg = unary.X
		}
		if ident, ok := arg.(*ast.Ident); ok {
			out = append(out, ident.Name)
		}
	}
	return out
}

func guard(files map[string]*ast.File, fset *token.FileSet) []string {
	g := &scopeGuard{fset: fset, scoping: scopingHelpers(files)}
	for path, file := range files {
		g.checkLiterals(path, file)
		if strings.HasPrefix(path, "internal/logfilter/") {
			continue
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				g.checkFunc(path, fn)
			}
		}
	}
	return g.findings
}

func parseTree(t *testing.T, fset *token.FileSet) map[string]*ast.File {
	t.Helper()
	files := map[string]*ast.File{}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		files[filepath.ToSlash(path)] = file
		return err
	})
	if err != nil {
		t.Fatalf("parse the API: %v", err)
	}
	return files
}

// TestEveryLogQueryIsScoped fails when code reads log_entries without going
// through internal/logfilter, whose every entry point filters on the caller's
// workspaces, or writes an entry without its workspace. Retention, the one
// job that deletes across workspaces by design, is the only raw SQL allowed.
func TestEveryLogQueryIsScoped(t *testing.T) {
	fset := token.NewFileSet()
	if findings := guard(parseTree(t, fset), fset); len(findings) > 0 {
		t.Fatalf("unscoped log_entries access:\n%s", strings.Join(findings, "\n"))
	}
}

// TestTheScopeGuardCatchesALeak runs the guard on the query shapes the
// tenancy audit found, so a guard that silently passes everything fails here.
func TestTheScopeGuardCatchesALeak(t *testing.T) {
	fset := token.NewFileSet()
	src := `package leak
func leak(db *gorm.DB, id int64) {
	var entry schemas.LogEntry
	db.WithContext(ctx).First(&entry, id)
	var page []schemas.LogEntry
	query := db.Model(&schemas.LogEntry{}).Where("app = ?", "x")
	query.Order("id").Find(&page)
	db.Raw("SELECT * FROM log_entries").Scan(&page)
	db.Create(&schemas.LogEntry{App: "x"})
}`
	file, err := parser.ParseFile(fset, "modules/leak/leak.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if findings := guard(map[string]*ast.File{"modules/leak/leak.go": file}, fset); len(findings) != 6 {
		t.Fatalf("want 6 findings (First, Model, Find, raw SQL, Scan, unscoped write), got %d:\n%s", len(findings), strings.Join(findings, "\n"))
	}
}
