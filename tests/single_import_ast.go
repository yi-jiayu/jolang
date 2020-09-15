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
			&ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: "\"fmt\"",
						},
					},
				},
			},
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
										Value: "\"Hello, World\\n\"",
									},
								},
							},
						},
					},
				},
			},
		},
		Imports: []*ast.ImportSpec{
			&ast.ImportSpec{
				Path: &ast.BasicLit{
					Kind:  token.STRING,
					Value: "\"fmt\"",
				},
			},
		},
	}

	printer.Fprint(os.Stdout, token.NewFileSet(), node)
}
