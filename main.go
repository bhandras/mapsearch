package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

func main() {
	cfg := &packages.Config{
		Mode:  packages.LoadSyntax, // Load syntax and type info
		Fset:  token.NewFileSet(),
		Dir:   ".",
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		log.Fatalf("failed to load packages: %v", err)
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			info := pkg.TypesInfo

			ast.Inspect(file, func(n ast.Node) bool {
				rangeStmt, ok := n.(*ast.RangeStmt)
				if !ok {
					return true
				}

				// Ensure key and value are identifiers
				keyIdent, ok1 := rangeStmt.Key.(*ast.Ident)
				valIdent, ok2 := rangeStmt.Value.(*ast.Ident)
				if !ok1 || !ok2 {
					return true
				}

				// Check if we're ranging over a map
				rangeExprType := info.TypeOf(rangeStmt.X)
				_, ok = rangeExprType.(*types.Map)
				if !ok {
					return true // not ranging over a map
				}

				for _, stmt := range rangeStmt.Body.List {
					assignStmt, ok := stmt.(*ast.AssignStmt)
					if !ok || len(assignStmt.Lhs) != 1 ||
						len(assignStmt.Rhs) != 1 {

						continue
					}

					// Check for destMap[key] = val
					indexExpr, ok := assignStmt.Lhs[0].(*ast.IndexExpr)
					if !ok {
						continue
					}

					// Check that key and value match loop vars
					indexKey, ok1 := indexExpr.Index.(*ast.Ident)
					rhsVal, ok2 := assignStmt.Rhs[0].(*ast.Ident)
					if !ok1 || !ok2 {
						continue
					}

					if indexKey.Name != keyIdent.Name ||
						rhsVal.Name != valIdent.Name {

						continue
					}

					// Check that destMap is also a map
					lhsObj := info.TypeOf(indexExpr.X)
					_, ok = lhsObj.(*types.Map)
					if !ok {
						continue
					}

					pos := cfg.Fset.Position(assignStmt.Pos())
					fmt.Printf("Map-to-map copy: %s:%d\n",
						pos.Filename, pos.Line)
				}

				return true
			})
		}
	}
}
