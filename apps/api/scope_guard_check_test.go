package main

import (
	"go/ast"
	"go/token"
	"strings"
)

func (g *scopeGuard) checkFunc(path string, fn *ast.FuncDecl) {
	vars := logEntryVars(fn)
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !terminal[sel.Sel.Name] {
			return true
		}
		for _, name := range argNames(call.Args) {
			if vars[name] && !g.scopedChain(sel.X, fn, 0) {
				g.report(path, call.Pos(), "a log_entries read that does not go through logfilter: "+sel.Sel.Name+"(&"+name+")")
			}
		}
		return true
	})
}

func (g *scopeGuard) checkLiterals(path string, file *ast.File) {
	inLogfilter := strings.HasPrefix(path, "internal/logfilter/")
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CompositeLit:
			if node.Type == nil || !isLogEntryType(node.Type) {
				return true
			}
			if _, isArray := node.Type.(*ast.ArrayType); isArray {
				return true
			}
			if len(node.Elts) > 0 && !hasKey(node, "WorkspaceID") {
				g.report(path, node.Pos(), "a schemas.LogEntry written without its WorkspaceID")
			}
		case *ast.CallExpr:
			for _, arg := range node.Args {
				if isEmptyLogEntry(arg) && !inLogfilter {
					g.report(path, arg.Pos(), "schemas.LogEntry{} as a query model outside internal/logfilter")
				}
			}
		case *ast.BasicLit:
			if node.Kind == token.STRING && strings.Contains(node.Value, "log_entries") && !rawAllowed(path) {
				g.report(path, node.Pos(), "raw SQL on log_entries outside the allowed packages")
			}
		}
		return true
	})
}

func isEmptyLogEntry(arg ast.Expr) bool {
	if unary, ok := arg.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		arg = unary.X
	}
	lit, ok := arg.(*ast.CompositeLit)
	return ok && lit.Type != nil && len(lit.Elts) == 0 && isLogEntryType(lit.Type)
}

func hasKey(lit *ast.CompositeLit, key string) bool {
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok && isIdent(kv.Key, key) {
			return true
		}
	}
	return false
}

func rawAllowed(path string) bool {
	for _, prefix := range rawTableAllowed {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func scopingHelpers(files map[string]*ast.File) map[string]bool {
	out := map[string]bool{}
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if ret, ok := n.(*ast.ReturnStmt); ok {
					for _, result := range ret.Results {
						if call, ok := result.(*ast.CallExpr); ok {
							if sel, ok := call.Fun.(*ast.SelectorExpr); ok && isIdent(sel.X, "logfilter") {
								out[fn.Name.Name] = true
							}
						}
					}
				}
				return true
			})
		}
	}
	return out
}
