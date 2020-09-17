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
							Value: "\"time\"",
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
								Fun: &ast.Ident{
									Name: "println",
								},
								Args: []ast.Expr{
									&ast.CallExpr{
										Fun: &ast.SelectorExpr{
											X: &ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X: &ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X: &ast.Ident{
																Name: "time",
															},
															Sel: &ast.Ident{
																Name: "Now",
															},
														},
													},
													Sel: &ast.Ident{
														Name: "Add",
													},
												},
												Args: []ast.Expr{
													&ast.SelectorExpr{
														X: &ast.Ident{
															Name: "time",
														},
														Sel: &ast.Ident{
															Name: "Second",
														},
													},
												},
											},
											Sel: &ast.Ident{
												Name: "Unix",
											},
										},
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
					Value: "\"time\"",
				},
			},
		},
	}

	printer.Fprint(os.Stdout, token.NewFileSet(), node)
}
