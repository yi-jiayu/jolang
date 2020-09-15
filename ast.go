package jo

import (
	"errors"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

func sexprsToAST(source string) (*ast.File, error) {
	file := &ast.File{}
	_, matched, err := SExprs()(source)
	if err != nil {
		return nil, err
	}
	sexprs := matched.([]interface{})
	pkgClause, sexprs := sexprs[0], sexprs[1:]
	pkg, err := getPackage(pkgClause)
	if err != nil {
		return nil, err
	}
	file.Name = &ast.Ident{
		Name: pkg,
	}
	var topLevelDecls []ast.Decl
	for _, sexpr := range sexprs {
		decl, err := newTopLevelDeclaration(sexpr)
		if err != nil {
			return nil, err
		}
		topLevelDecls = append(topLevelDecls, decl)
	}
	file.Decls = topLevelDecls
	return file, nil
}

// Compile compiles Jo source code into Go source code.
func Compile(source string) (string, error) {
	node, err := sexprsToAST(source)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	err = printer.Fprint(&b, token.NewFileSet(), node)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func getPackage(expr interface{}) (string, error) {
	pkgClause, ok := expr.([]interface{})
	if !ok || len(pkgClause) < 2 || pkgClause[0] != "package" {
		return "", errors.New("not a package clause")
	}
	return pkgClause[1].(string), nil
}

func newTopLevelDeclaration(expr interface{}) (ast.Decl, error) {
	exprs, ok := expr.([]interface{})
	if !ok || len(exprs) < 1 || exprs[0] != "func" {
		return nil, errors.New("not a top level declaration")
	}
	_, exprs = exprs[0], exprs[1:] // "func" keyword
	name, exprs := exprs[0], exprs[1:]
	_, stmts := exprs[0], exprs[1:] // argument list and statements
	var body []ast.Stmt
	for _, stmt := range stmts {
		s, err := newStatement(stmt)
		if err != nil {
			return nil, err
		}
		body = append(body, s)
	}
	return &ast.FuncDecl{
		Name: &ast.Ident{
			Name: name.(string),
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
		Body: &ast.BlockStmt{
			List: body,
		},
	}, nil
}

func newStatement(expr interface{}) (ast.Stmt, error) {
	exprs := expr.([]interface{})
	fname := exprs[0].(string)
	var args []ast.Expr
	for _, e := range exprs[1:] {
		args = append(args, &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + e.(string) + `"`,
		})
	}
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.Ident{
				Name: fname,
			},
			Args: args,
		},
	}, nil
}
