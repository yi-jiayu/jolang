package jo

import (
	"fmt"
	"go/ast"
	"strings"
)

func sprint(node ast.Node) string {
	switch n := node.(type) {
	case *ast.Ident:
		return sprintIdent(n)
	case *ast.BasicLit:
		return sprintBasicLit(n)
	case *ast.SelectorExpr:
		return sprintSelectorExpr(n)
	}
	return ""
}

func sprintCallExpr(expr *ast.CallExpr) string {
	list := []string{sprint(expr.Fun)}
	for _, arg := range expr.Args {
		list = append(list, sprint(arg))
	}
	return fmt.Sprintf("(%s)", strings.Join(list, " "))

}

func sprintSelectorExpr(expr *ast.SelectorExpr) string {
	return fmt.Sprintf("%s.%s", sprint(expr.X), sprintIdent(expr.Sel))
}

func sprintIdent(ident *ast.Ident) string {
	return ident.Name
}

func sprintBasicLit(lit *ast.BasicLit) string {
	return lit.Value
}
