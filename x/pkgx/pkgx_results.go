package pkg

import (
	"go/ast"
	"go/types"
)

type TypeAndValueExpr struct {
	ast.Expr
	types.TypeAndValue
}

type Results map[int][]TypeAndValueExpr

func IsContainsReturn(n ast.Node) (ok bool) {
	ast.Inspect(n, func(node ast.Node) bool {
		_, ok = n.(*ast.ReturnStmt)
		return true
	})
	return
}
