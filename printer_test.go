package jo

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_sprintSelectorExpr(t *testing.T) {
	expr := &ast.SelectorExpr{
		X: &ast.Ident{
			Name: "fmt",
		},
		Sel: &ast.Ident{
			Name: "Printf",
		},
	}
	s := sprintSelectorExpr(expr)
	assert.Equal(t, "fmt.Printf", s)
}

func Test_sprintCallExpr(t *testing.T) {
	expr := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.Ident{
				Name: "fmt",
			},
			Sel: &ast.Ident{
				Name: "Printf",
			},
		},
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: "\"string: %q, integer: %d\\n\"",
			},
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: "\"hello\"",
			},
			&ast.BasicLit{
				Kind:  token.INT,
				Value: "1",
			},
		},
	}
	s := sprintCallExpr(expr)
	assert.Equal(t, `(fmt.Printf "string: %q, integer: %d\n" "hello" 1)`, s)
}
