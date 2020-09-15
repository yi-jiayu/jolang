package main

import (
	"go/ast"
	"go/printer"
	"go/token"
	"os"
)

func main() {
	node := &ast.File{
		Package: 1,
		Name: &ast.Ident{
			Name: "main",
		},
		Decls: []ast.Decl{
			&ast.FuncDecl{
				Name: &ast.Ident{
					Name: "main",
				},
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.CallExpr{
								Fun: &ast.Ident{
									Name: "println",
								},
								Args: []ast.Expr{
									&ast.BasicLit{
										Kind:  token.STRING,
										Value: "\"Hello, World\"",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	printer.Fprint(os.Stdout, token.NewFileSet(), node)
}
