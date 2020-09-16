package jo

import (
	"go/ast"
	"go/token"
	"strconv"
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
	t.Run("without imports", func(t *testing.T) {
		const input = `(package main)

(func main () (println "Hello, World"))`
		_, matched, err := SourceFile(input)
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
	})
	t.Run("with imports", func(t *testing.T) {
		const input = `(package main)

(import "fmt")

(func main () (fmt.Println 1))`
		_, matched, err := SourceFile(input)
		assert.Equal(t, &ast.File{
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
										X:   ident("fmt"),
										Sel: ident("Println"),
									},
									Args: []ast.Expr{
										&ast.BasicLit{
											Kind:  token.INT,
											Value: "1",
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
	})
}

func TestSExpr(t *testing.T) {
	p := Parenthesized(OneOrMore(WhitespaceWrap(Identifier)))
	remaining, matched, err := p("(hello world)")
	assert.Equal(t, "", remaining)
	assert.Equal(t, []interface{}{"hello", "world"}, matched)
	assert.NoError(t, err)
}

func Test_callExpr_Parse(t *testing.T) {
	t.Run("literal arguments", func(t *testing.T) {
		_, matched, err := CallExpr.Parse(`(println "Hello, World")`)
		assert.Equal(t, &ast.CallExpr{
			Fun:  ident("println"),
			Args: []ast.Expr{strLit(`"Hello, World"`)},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("no arguments", func(t *testing.T) {
		_, matched, err := CallExpr.Parse(`(f)`)
		assert.Equal(t, &ast.CallExpr{
			Fun: ident("f"),
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("nested call expressions", func(t *testing.T) {
		_, matched, err := CallExpr.Parse(`(println "Hello" (fmt.Sprint "World"))`)
		assert.Equal(t, &ast.CallExpr{
			Fun: ident("println"),
			Args: []ast.Expr{
				strLit(`"Hello"`),
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: "fmt",
						},
						Sel: &ast.Ident{
							Name: "Sprint",
						},
					},
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: "\"World\"",
						},
					},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
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

func TestList(t *testing.T) {
	p := Sequence(Literal("hello"), Literal(" "), Literal("world"))
	remaining, matched, err := p("hello world!")
	assert.Equal(t, "!", remaining)
	assert.Equal(t, []interface{}{"hello", " ", "world"}, matched)
	assert.NoError(t, err)
}

func TestImportDecl(t *testing.T) {
	p := ImportDecl()
	t.Run("single import", func(t *testing.T) {
		_, matched, err := p(`(import "fmt")`)
		assert.Equal(t, &ast.GenDecl{
			Tok: token.IMPORT,
			Specs: []ast.Spec{
				&ast.ImportSpec{
					Path: &ast.BasicLit{
						Kind:  token.STRING,
						Value: "\"fmt\"",
					},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("grouped import", func(t *testing.T) {
		_, matched, err := p(`(import "fmt" "log")`)
		assert.Equal(t, &ast.GenDecl{
			Tok: token.IMPORT,
			Specs: []ast.Spec{
				&ast.ImportSpec{
					Path: &ast.BasicLit{
						Kind:  token.STRING,
						Value: "\"fmt\"",
					},
				},
				&ast.ImportSpec{
					Path: &ast.BasicLit{
						Kind:  token.STRING,
						Value: "\"log\"",
					},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
}

func TestQualifiedIdent(t *testing.T) {
	_, matched, err := QualifiedIdent("fmt.Println")
	assert.Equal(t, &ast.SelectorExpr{
		X: &ast.Ident{
			Name: "fmt",
		},
		Sel: &ast.Ident{
			Name: "Println",
		},
	}, matched)
	assert.NoError(t, err)
}

func Test_identifier(t *testing.T) {
	t.Run("unqualified", func(t *testing.T) {
		_, matched, err := identifier("println")
		assert.Equal(t, ident("println"), matched)
		assert.NoError(t, err)
	})
	t.Run("qualified indentifier", func(t *testing.T) {
		_, matched, err := identifier("fmt.Println")
		assert.Equal(t, &ast.SelectorExpr{
			X: &ast.Ident{
				Name: "fmt",
			},
			Sel: &ast.Ident{
				Name: "Println",
			},
		}, matched)
		assert.NoError(t, err)
	})
}

func intLit(v int) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.INT,
		Value: strconv.Itoa(v),
	}
}

func Test_binaryExpr_Parse(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		_, matched, err := BinaryExpr.Parse(`(+ 1 2)`)
		assert.Equal(t, &ast.BinaryExpr{
			X:  intLit(1),
			Op: token.ADD,
			Y:  intLit(2),
		}, matched)
		assert.NoError(t, err)
	})
}
