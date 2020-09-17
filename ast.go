package jo

import (
	"go/ast"
)

func Parse(input string) (*ast.File, error) {
	_, node, err := SourceFile(NewSource(input))
	if err != nil {
		return nil, err
	}
	return node.(*ast.File), nil
}

func newIdent(v string) *ast.Ident {
	return &ast.Ident{
		Name: v,
	}
}

func newSelectorExpr(x, sel interface{}) *ast.SelectorExpr {
	var expr ast.SelectorExpr
	switch v := x.(type) {
	case ast.Expr:
		expr.X = v
	case string:
		expr.X = newIdent(v)
	}
	switch v := sel.(type) {
	case *ast.Ident:
		expr.Sel = v
	case string:
		expr.Sel = newIdent(v)
	}
	return &expr
}

func newCallExpr(fun interface{}, args ...ast.Expr) *ast.CallExpr {
	expr := &ast.CallExpr{
		Args: args,
	}
	switch v := fun.(type) {
	case ast.Expr:
		expr.Fun = v
	case string:
		expr.Fun = newIdent(v)
	}
	return expr
}
