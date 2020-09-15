package jo

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Literal(t *testing.T) {
	parseJoe := Literal("Hello Joe!")
	{
		remaining, matched, err := parseJoe("Hello Joe!")
		assert.Empty(t, remaining)
		assert.Equal(t, "Hello Joe!", matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := parseJoe("Hello Joe! Hello Robert!")
		assert.Equal(t, " Hello Robert!", remaining)
		assert.Equal(t, "Hello Joe!", matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := parseJoe("Hello Mike!")
		assert.EqualError(t, err, "wanted a literal \"Hello Joe!\", got: \"Hello Mike!\"")
	}
}

func Test_Identifier(t *testing.T) {
	{
		remaining, matched, err := Identifier("i_am_an_identifier")
		assert.Empty(t, remaining)
		assert.Equal(t, "i_am_an_identifier", matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := Identifier("not entirely an identifier")
		assert.Equal(t, " entirely an identifier", remaining)
		assert.Equal(t, "not", matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := Identifier("!not at all an identifier")
		assert.EqualError(t, err, "!not at all an identifier")
	}
}

func Test_Pair(t *testing.T) {
	tagOpener := Pair(Literal("<"), Identifier)
	{
		remaining, matched, err := tagOpener("<element/>")
		assert.Equal(t, "/>", remaining)
		assert.Equal(t, MatchedPair{Left: "<", Right: "element"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, _, err := tagOpener("oops")
		assert.Equal(t, "oops", remaining)
		assert.EqualError(t, err, "wanted a literal \"<\", got: \"oops\"")
	}
	{
		remaining, _, err := tagOpener("<!oops")
		assert.Equal(t, "<!oops", remaining)
		assert.EqualError(t, err, "!oops")
	}
}

func Test_Right(t *testing.T) {
	tagOpener := Right(Literal("<"), Identifier)
	{
		remaining, matched, err := tagOpener("<element/>")
		assert.Equal(t, "/>", remaining)
		assert.Equal(t, "element", matched)
		assert.NoError(t, err)
	}
}

func Test_OneOrMore(t *testing.T) {
	p := OneOrMore(Literal("ha"))
	{
		remaining, matched, err := p("hahaha")
		assert.Empty(t, remaining)
		assert.Equal(t, []interface{}{"ha", "ha", "ha"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("hahaha ahah")
		assert.Equal(t, " ahah", remaining)
		assert.Equal(t, []interface{}{"ha", "ha", "ha"}, matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := p("ahah")
		assert.EqualError(t, err, "wanted a literal \"ha\", got: \"ahah\"")
	}
	{
		_, _, err := p("")
		assert.EqualError(t, err, "wanted a literal \"ha\", got: \"\"")
	}
}

func Test_ZeroOrMore(t *testing.T) {
	p := ZeroOrMore(Literal("ha"))
	{
		remaining, matched, err := p("hahaha")
		assert.Empty(t, remaining)
		assert.Equal(t, []interface{}{"ha", "ha", "ha"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("ahah")
		assert.Equal(t, "ahah", remaining)
		assert.Empty(t, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("")
		assert.Equal(t, "", remaining)
		assert.Empty(t, matched)
		assert.NoError(t, err)
	}
}

func Test_Pred(t *testing.T) {
	p := Pred(AnyChar, func(matched interface{}) bool {
		return matched == 'o'
	})
	{
		remaining, matched, err := p("omg")
		assert.Equal(t, "mg", remaining)
		assert.Equal(t, 'o', matched)
		assert.NoError(t, err)
	}
	{
		remaining, _, err := p("lol")
		assert.Equal(t, "lol", remaining)
		assert.EqualError(t, err, "lol")
	}
}

func Test_QuotedString(t *testing.T) {
	p := QuotedString()
	remaining, matched, err := p(`"Hello Joe!"`)
	assert.Equal(t, "", remaining)
	assert.Equal(t, "Hello Joe!", matched)
	assert.NoError(t, err)
}

func Test_Choice(t *testing.T) {
	p := Choice(Literal("package"), Literal("func"))
	{
		remaining, matched, err := p("package main")
		assert.Equal(t, " main", remaining)
		assert.Equal(t, "package", matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("func main")
		assert.Equal(t, " main", remaining)
		assert.Equal(t, "func", matched)
		assert.NoError(t, err)
	}
	{
		remaining, _, err := p("import \"fmt\"")
		assert.Equal(t, `import "fmt"`, remaining)
		assert.Error(t, err)
	}
}

func ident(v string) *ast.Ident {
	return &ast.Ident{
		Name: v,
	}
}

func strLit(v string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.STRING,
		Value: v,
	}
}

func Test_SExpr(t *testing.T) {
	p := SExpr()
	{
		remaining, matched, err := p(`()`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p(`(import main)`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{token.IMPORT, ident("main")}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p(`(println "Hello, World")`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{ident("println"), strLit("\"Hello, World\"")}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p(`(	println  "Hello, World" )`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{ident("println"), strLit("\"Hello, World\"")}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, _, err := p(`println "Hello, World"`)
		assert.Equal(t, `println "Hello, World"`, remaining)
		assert.EqualError(t, err, "wanted a literal \"(\", got: \"println \\\"Hello, World\\\"\"")
	}
	t.Run("recursion", func(t *testing.T) {
		remaining, matched, err := p(`(func main ())`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{token.FUNC, ident("main"), []interface{}{}}, matched)
		assert.NoError(t, err)
	})
}

func TestPackageClause(t *testing.T) {
	p := PackageClause()
	remaining, matched, err := p("package main")
	assert.Equal(t, "", remaining)
	assert.Equal(t, "main", matched)
	assert.NoError(t, err)
}

func Test_decimalLit(t *testing.T) {
	p := decimalLit()
	{
		remaining, matched, err := p("0 aoeu")
		assert.Equal(t, " aoeu", remaining)
		assert.Equal(t, &ast.BasicLit{Kind: token.INT, Value: "0"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("12340 aoeu")
		assert.Equal(t, " aoeu", remaining)
		assert.Equal(t, &ast.BasicLit{Kind: token.INT, Value: "12340"}, matched)
		assert.NoError(t, err)
	}
}

func Test_stringLit(t *testing.T) {
	p := stringLit()
	remaining, matched, err := p(`"Hello, World"`)
	assert.Equal(t, "", remaining)
	assert.Equal(t, &ast.BasicLit{Kind: token.STRING, Value: "\"Hello, World\""}, matched)
	assert.NoError(t, err)
}

func TestSourceFile(t *testing.T) {
	p := SourceFile()
	const input = `(package main)

(func main () (println "Hello, World"))`
	_, matched, err := p(input)
	assert.Equal(t, &ast.File{
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
	}, matched)
	assert.NoError(t, err)
}

func TestSExpr2(t *testing.T) {
	p := SExpr2(OneOrMore(WhitespaceWrap(Identifier)))
	remaining, matched, err := p("(hello world)")
	assert.Equal(t, "", remaining)
	assert.Equal(t, []interface{}{"hello", "world"}, matched)
	assert.NoError(t, err)
}

func TestCallExpr(t *testing.T) {
	p := CallExpr()
	_, matched, err := p(`(println "Hello, World")`)
	assert.Equal(t, &ast.CallExpr{
		Fun:  ident("println"),
		Args: []ast.Expr{strLit(`"Hello, World"`)},
	}, matched)
	assert.NoError(t, err)
}

func TestFunctionDecl(t *testing.T) {
	p := FunctionDecl()
	_, matched, err := p(`(func main () (println "Hello, World"))`)
	assert.Equal(t, &ast.FuncDecl{
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
	}, matched)
	assert.NoError(t, err)

}
