package jo

import (
	"go/ast"
	"go/token"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func stringParser(p Parser) func(input string) (remaining Source, matched interface{}, err error) {
	return func(input string) (remaining Source, matched interface{}, err error) {
		return p.Parse(NewSource(input))
	}
}

func Test_Literal(t *testing.T) {
	parseJoe := stringParser(Literal("Hello Joe!"))
	{
		remaining, matched, err := parseJoe("Hello Joe!")
		assert.Equal(t, Source{Offset: 10}, remaining)
		assert.Equal(t, "Hello Joe!", matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := parseJoe("Hello Joe! Hello Robert!")
		assert.Equal(t, Source{Content: " Hello Robert!", Offset: 10}, remaining)
		assert.Equal(t, "Hello Joe!", matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := parseJoe("Hello Mike!")
		assert.Equal(t, &ParseError{Offset: 0, Message: "wanted a literal \"Hello Joe!\", got: \"H\""}, err)
	}
}

func Test_Identifier(t *testing.T) {
	parse := stringParser(Identifier)
	{
		remaining, matched, err := parse("i_am_an_identifier")
		assert.Equal(t, Source{Offset: 18}, remaining)
		assert.Equal(t, "i_am_an_identifier", matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := parse("not entirely an identifier")
		assert.Equal(t, Source{Content: " entirely an identifier", Offset: 3}, remaining)
		assert.Equal(t, "not", matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := parse("!not at all an identifier")
		assert.Equal(t, &ParseError{Offset: 0, Message: "wanted identifier, got '!'"}, err)
	}
	t.Run("blank identifier", func(t *testing.T) {
		_, matched, err := parse("_")
		assert.Equal(t, "_", matched)
		assert.NoError(t, err)
	})
}

func Test_Pair(t *testing.T) {
	tagOpener := stringParser(Pair(Literal("<"), Identifier))
	{
		remaining, matched, err := tagOpener("<element/>")
		assert.Equal(t, Source{Content: "/>", Offset: 8}, remaining)
		assert.Equal(t, MatchedPair{Left: "<", Right: "element"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, _, err := tagOpener("oops")
		assert.Equal(t, Source{Content: "oops", Offset: 0}, remaining)
		assert.Equal(t, &ParseError{Offset: 0, Message: `wanted a literal "<", got: "o"`}, err)
	}
	{
		remaining, _, err := tagOpener("<!oops")
		assert.Equal(t, Source{Content: "<!oops", Offset: 0}, remaining)
		assert.Equal(t, &ParseError{Offset: 1, Message: "wanted identifier, got '!'"}, err)
	}
}

func Test_Right(t *testing.T) {
	tagOpener := stringParser(Right(Literal("<"), Identifier))
	{
		remaining, matched, err := tagOpener("<element/>")
		assert.Equal(t, Source{Content: "/>", Offset: 8}, remaining)
		assert.Equal(t, "element", matched)
		assert.NoError(t, err)
	}
}

func Test_OneOrMore(t *testing.T) {
	p := stringParser(OneOrMore(Literal("ha")))
	{
		remaining, matched, err := p("hahaha")
		assert.Equal(t, Source{Offset: 6}, remaining)
		assert.Equal(t, []interface{}{"ha", "ha", "ha"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("hahaha ahah")
		assert.Equal(t, Source{Content: " ahah", Offset: 6}, remaining)
		assert.Equal(t, []interface{}{"ha", "ha", "ha"}, matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := p("ahah")
		assert.Equal(t, &ParseError{Offset: 0, Message: `wanted a literal "ha", got: "a"`}, err)
	}
	{
		_, _, err := p("")
		assert.Equal(t, &ParseError{Offset: 0, Message: "wanted a literal \"ha\", got: \"\""}, err)
	}
}

func Test_ZeroOrMore(t *testing.T) {
	p := stringParser(ZeroOrMore(Literal("ha")))
	{
		remaining, matched, err := p("hahaha")
		assert.Equal(t, Source{Offset: 6}, remaining)
		assert.Equal(t, []interface{}{"ha", "ha", "ha"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("ahah")
		assert.Equal(t, Source{Content: "ahah"}, remaining)
		assert.Empty(t, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("")
		assert.Equal(t, Source{}, remaining)
		assert.Empty(t, matched)
		assert.NoError(t, err)
	}
}

func Test_Pred(t *testing.T) {
	p := stringParser(Pred(AnyChar, func(matched interface{}) bool {
		return matched == 'o'
	}))
	{
		remaining, matched, err := p("omg")
		assert.Equal(t, Source{Content: "mg", Offset: 1}, remaining)
		assert.Equal(t, 'o', matched)
		assert.NoError(t, err)
	}
	{
		remaining, _, err := p("lol")
		assert.Equal(t, Source{Content: "lol"}, remaining)
		assert.Equal(t, &ParseError{Message: "predicate failed"}, err)
	}
}

func Test_QuotedString(t *testing.T) {
	p := stringParser(QuotedString())
	remaining, matched, err := p(`"Hello Joe!"`)
	assert.Equal(t, Source{Offset: 12}, remaining)
	assert.Equal(t, "Hello Joe!", matched)
	assert.NoError(t, err)
}

func Test_Choice(t *testing.T) {
	p := stringParser(Choice(Literal("package"), Literal("func")))
	{
		remaining, matched, err := p("package main")
		assert.Equal(t, Source{Content: " main", Offset: 7}, remaining)
		assert.Equal(t, "package", matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("func main")
		assert.Equal(t, Source{Content: " main", Offset: 4}, remaining)
		assert.Equal(t, "func", matched)
		assert.NoError(t, err)
	}
	{
		remaining, _, err := p("import \"fmt\"")
		assert.Equal(t, Source{Content: "import \"fmt\""}, remaining)
		assert.Error(t, err)
	}
}

func strLit(v string) *ast.BasicLit {
	return &ast.BasicLit{
		Kind:  token.STRING,
		Value: v,
	}
}

func Test_decimalLit(t *testing.T) {
	p := stringParser(decimalLit())
	{
		remaining, matched, err := p("0 aoeu")
		assert.Equal(t, Source{Content: " aoeu", Offset: 1}, remaining)
		assert.Equal(t, &ast.BasicLit{Kind: token.INT, Value: "0"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("12340 aoeu")
		assert.Equal(t, Source{Content: " aoeu", Offset: 5}, remaining)
		assert.Equal(t, &ast.BasicLit{Kind: token.INT, Value: "12340"}, matched)
		assert.NoError(t, err)
	}
}

func Test_stringLit(t *testing.T) {
	p := stringParser(stringLit())
	remaining, matched, err := p(`"Hello, World"`)
	assert.Equal(t, Source{Offset: 14}, remaining)
	assert.Equal(t, &ast.BasicLit{Kind: token.STRING, Value: "\"Hello, World\""}, matched)
	assert.NoError(t, err)
}

func TestSourceFile(t *testing.T) {
	t.Run("without imports", func(t *testing.T) {
		const input = `(package main)

(func main () (println "Hello, World"))`
		_, matched, err := SourceFile(NewSource(input))
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
		_, matched, err := SourceFile(NewSource(input))
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
										X:   ast.NewIdent("fmt"),
										Sel: ast.NewIdent("Println"),
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
	p := stringParser(Parenthesized(OneOrMore(WhitespaceWrap(Identifier))))
	remaining, matched, err := p("(hello world)")
	assert.Equal(t, Source{Offset: 13}, remaining)
	assert.Equal(t, []interface{}{"hello", "world"}, matched)
	assert.NoError(t, err)
}

func Test_callExpr_Parse(t *testing.T) {
	parse := stringParser(CallExpr)
	t.Run("literal arguments", func(t *testing.T) {
		_, matched, err := parse(`(println "Hello, World")`)
		assert.Equal(t, &ast.CallExpr{
			Fun:  ast.NewIdent("println"),
			Args: []ast.Expr{strLit(`"Hello, World"`)},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("no arguments", func(t *testing.T) {
		_, matched, err := parse(`(f)`)
		assert.Equal(t, &ast.CallExpr{
			Fun: ast.NewIdent("f"),
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("nested call expressions", func(t *testing.T) {
		_, matched, err := parse(`(println "Hello" (fmt.Sprint "World"))`)
		assert.Equal(t, &ast.CallExpr{
			Fun: ast.NewIdent("println"),
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
	parse := stringParser(FunctionDecl)
	_, matched, err := parse(`(func main () (println "Hello, World"))`)
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
	p := stringParser(Sequence(Literal("hello"), Literal(" "), Literal("world")))
	remaining, matched, err := p("hello world!")
	assert.Equal(t, Source{Content: "!", Offset: 11}, remaining)
	assert.Equal(t, []interface{}{"hello", " ", "world"}, matched)
	assert.NoError(t, err)
}

func TestImportDecl(t *testing.T) {
	parse := stringParser(ImportDecl)
	t.Run("single import", func(t *testing.T) {
		_, matched, err := parse(`(import "fmt")`)
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
		_, matched, err := parse(`(import "fmt" "log")`)
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
	parse := stringParser(QualifiedIdent)
	_, matched, err := parse("fmt.Println")
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

func TestOperandName(t *testing.T) {
	parse := stringParser(OperandName)
	t.Run("unqualified", func(t *testing.T) {
		_, matched, err := parse("println")
		assert.Equal(t, ast.NewIdent("println"), matched)
		assert.NoError(t, err)
	})
	t.Run("qualified indentifier", func(t *testing.T) {
		_, matched, err := parse("fmt.Println")
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
	parse := stringParser(BinaryExpr)
	t.Run("single", func(t *testing.T) {
		_, matched, err := parse(`(+ 1 2)`)
		assert.Equal(t, &ast.BinaryExpr{
			X:  intLit(1),
			Op: token.ADD,
			Y:  intLit(2),
		}, matched)
		assert.NoError(t, err)
	})
}

func Test_selector_Parse(t *testing.T) {
	parse := stringParser(Selector)
	t.Run("field access", func(t *testing.T) {
		_, matched, err := parse(`(sel myStruct Outer Middle Inner)`)
		assert.Equal(t, &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X: &ast.SelectorExpr{
					X:   ast.NewIdent("myStruct"),
					Sel: ast.NewIdent("Outer"),
				},
				Sel: ast.NewIdent("Middle"),
			},
			Sel: ast.NewIdent("Inner"),
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("function calls", func(t *testing.T) {
		_, matched, err := parse(`(sel time (Now) (Add time.Second))`)
		assert.Equal(t, &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("time"),
						Sel: ast.NewIdent("Now"),
					},
				},
				Sel: ast.NewIdent("Add"),
			},
			Args: []ast.Expr{newSelectorExpr("time", "Second")},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("sel on expr", func(t *testing.T) {
		_, matched, err := parse(`(sel (now) (Unix))`)
		assert.Equal(t, &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.CallExpr{
					Fun: &ast.Ident{Name: "now"},
				},
				Sel: &ast.Ident{
					Name: "Unix",
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
}

func Test_structType_Parse(t *testing.T) {
	parse := stringParser(StructType)
	t.Run("simple", func(t *testing.T) {
		_, matched, err := parse(`(struct (Field1 int) (Field2 string))`)
		assert.Equal(t, &ast.StructType{
			Fields: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{{Name: "Field1"}},
						Type:  &ast.Ident{Name: "int"},
					},
					{
						Names: []*ast.Ident{{Name: "Field2"}},
						Type:  &ast.Ident{Name: "string"},
					},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
}

func Test_typeDecl_Parse(t *testing.T) {
	parse := stringParser(TypeDecl)
	t.Run("struct", func(t *testing.T) {
		_, matched, err := parse(`(type MyStruct (struct (Field string)))`)
		assert.Equal(t, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: &ast.Ident{Name: "MyStruct"},
					Type: &ast.StructType{
						Fields: &ast.FieldList{
							List: []*ast.Field{
								{
									Names: []*ast.Ident{{Name: "Field"}},
									Type:  &ast.Ident{Name: "string"},
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

func TestSource_Advance(t *testing.T) {
	s := NewSource("Hello")
	s = s.Advance(3)
	assert.Equal(t, Source{"lo", 3}, s)
	s = s.Advance(2)
	assert.Equal(t, Source{Content: "", Offset: 5}, s)
	s = s.Advance(1)
	assert.Equal(t, Source{Content: "", Offset: 5}, s)
}

func Test__decimalFloatLit_Parse(t *testing.T) {
	parse := stringParser(decimalFloatLit)
	_, matched, err := parse("0.1")
	assert.Equal(t, &ast.BasicLit{
		Kind:  token.FLOAT,
		Value: "0.1",
	}, matched)
	assert.NoError(t, err)
}

func TestIfStmt(t *testing.T) {
	parse := stringParser(IfStmt)
	t.Run("identifier cond", func(t *testing.T) {
		_, matched, err := parse(`(if true (println "true"))`)
		assert.Equal(t, &ast.IfStmt{
			Cond: ast.NewIdent("true"),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: newCallExpr("println", strLit(`"true"`))},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("expr cond", func(t *testing.T) {
		_, matched, err := parse(`(if (= 2 2) (println "true"))`)
		assert.Equal(t, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  intLit(2),
				Op: token.EQL,
				Y:  intLit(2),
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: newCallExpr("println", strLit(`"true"`))},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("do block", func(t *testing.T) {
		_, matched, err := parse(`(if true (do (println true) (println false)))`)
		assert.Equal(t, &ast.IfStmt{
			Cond: ast.NewIdent("true"),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: newCallExpr("println", ast.NewIdent("true"))},
					&ast.ExprStmt{X: newCallExpr("println", ast.NewIdent("false"))},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("else block", func(t *testing.T) {
		_, matched, err := parse(`(if true (println "true") (println "false"))`)
		assert.Equal(t, &ast.IfStmt{
			Cond: ast.NewIdent("true"),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: newCallExpr("println", strLit(`"true"`))},
				},
			},
			Else: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: newCallExpr("println", strLit(`"false"`))},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("else block with do", func(t *testing.T) {
		_, matched, err := parse(`(if true (println "true") (do (println "false") (println "false")))`)
		assert.Equal(t, &ast.IfStmt{
			Cond: ast.NewIdent("true"),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: newCallExpr("println", strLit(`"true"`))},
				},
			},
			Else: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: newCallExpr("println", strLit(`"false"`))},
					&ast.ExprStmt{X: newCallExpr("println", strLit(`"false"`))},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
}

func TestDoExpr(t *testing.T) {
	parse := stringParser(DoExpr)
	t.Run("empty", func(t *testing.T) {
		_, matched, err := parse(`(do)`)
		assert.Equal(t, &ast.BlockStmt{
			List: []ast.Stmt{},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("one expr", func(t *testing.T) {
		_, matched, err := parse(`(do (println true))`)
		assert.Equal(t, &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{X: newCallExpr("println", ast.NewIdent("true"))},
			},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("two expr", func(t *testing.T) {
		_, matched, err := parse(`(do (println true) (println false))`)
		assert.Equal(t, &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{X: newCallExpr("println", ast.NewIdent("true"))},
				&ast.ExprStmt{X: newCallExpr("println", ast.NewIdent("false"))},
			},
		}, matched)
		assert.NoError(t, err)
	})
}

func Test_statementList_Parse(t *testing.T) {
	parse := stringParser(StatementList)
	_, matched, err := parse(`(println 1) (if true (println 2))`)
	assert.Equal(t, []ast.Stmt{
		&ast.ExprStmt{X: newCallExpr("println", intLit(1))},
		&ast.IfStmt{
			Cond: ast.NewIdent("true"),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: newCallExpr("println", intLit(2))},
				},
			},
		},
	}, matched)
	assert.NoError(t, err)
}

func TestIdentifierList(t *testing.T) {
	parse := stringParser(IdentifierList)
	t.Run("single", func(t *testing.T) {
		_, matched, err := parse(`a`)
		assert.NoError(t, err)
		assert.Equal(t, []ast.Expr{
			ast.NewIdent("a"),
		}, matched)
	})
	t.Run("multiple", func(t *testing.T) {
		_, matched, err := parse(`(a b c)`)
		assert.NoError(t, err)
		assert.Equal(t, []ast.Expr{
			ast.NewIdent("a"),
			ast.NewIdent("b"),
			ast.NewIdent("c"),
		}, matched)
	})
}

func TestExpressionList(t *testing.T) {
	parse := stringParser(ExpressionList)
	t.Run("single ident", func(t *testing.T) {
		_, matched, err := parse(`a`)
		assert.NoError(t, err)
		assert.Equal(t, []ast.Expr{
			ast.NewIdent("a"),
		}, matched)
	})
	t.Run("single expression", func(t *testing.T) {
		_, matched, err := parse(`((+ 1 2))`)
		assert.NoError(t, err)
		assert.Equal(t, []ast.Expr{
			&ast.BinaryExpr{
				X:  intLit(1),
				Op: token.ADD,
				Y:  intLit(2),
			},
		}, matched)
	})
	t.Run("multiple idents", func(t *testing.T) {
		_, matched, err := parse(`(a b c)`)
		assert.NoError(t, err)
		assert.Equal(t, []ast.Expr{
			ast.NewIdent("a"),
			ast.NewIdent("b"),
			ast.NewIdent("c"),
		}, matched)
	})
	t.Run("list with single expression", func(t *testing.T) {
		_, matched, err := parse(`((+ 1 2))`)
		assert.NoError(t, err)
		assert.Equal(t, []ast.Expr{
			&ast.BinaryExpr{
				X:  intLit(1),
				Op: token.ADD,
				Y:  intLit(2),
			},
		}, matched)
	})
	t.Run("multiple expressions", func(t *testing.T) {
		_, matched, err := parse(`((+ 1 2) (r.ReadString '\n'))`)
		assert.NoError(t, err)
		assert.Equal(t, []ast.Expr{
			&ast.BinaryExpr{
				X:  intLit(1),
				Op: token.ADD,
				Y:  intLit(2),
			},
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("r"),
					Sel: ast.NewIdent("ReadString"),
				},
				Args: []ast.Expr{&ast.BasicLit{
					Kind:  token.CHAR,
					Value: `'\n'`,
				}},
			},
		}, matched)

	})
}

func TestDefine(t *testing.T) {
	parse := stringParser(Define)
	t.Run("single variable", func(t *testing.T) {
		_, matched, err := parse(`(define x 1)`)
		assert.Equal(t, &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("x")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{intLit(1)},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("multiple variables", func(t *testing.T) {
		_, matched, err := parse(`(define (x y) (1 2))`)
		assert.Equal(t, &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("x"), ast.NewIdent("y")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{intLit(1), intLit(2)},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("function call", func(t *testing.T) {
		_, matched, err := parse(`(define (text _) ((r.ReadString '\n')))`)
		assert.Equal(t, &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("text"), ast.NewIdent("_")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: newSelectorExpr("r", "ReadString"),
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.CHAR,
							Value: `'\n'`,
						},
					},
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
}

func TestUnaryExpr(t *testing.T) {
	parse := stringParser(UnaryExpr)
	t.Run("single", func(t *testing.T) {
		_, matched, err := parse(`&x`)
		assert.Equal(t, &ast.UnaryExpr{
			Op: token.AND,
			X:  ast.NewIdent("x"),
		}, matched)
		assert.NoError(t, err)
	})
}

func TestDeclStmt(t *testing.T) {
	parse := stringParser(DeclStmt)
	_, matched, err := parse(`(var x int)`)
	assert.Equal(t, &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdent("x")},
					Type:  ast.NewIdent("int"),
				},
			},
		},
	}, matched)
	assert.NoError(t, err)
}

func Test_escapedChar(t *testing.T) {
	parse := stringParser(escapedChar)
	_, matched, err := parse(`\a`)
	assert.Equal(t, `\a`, matched)
	assert.NoError(t, err)
}

func TestRuneLit(t *testing.T) {
	parse := stringParser(RuneLit)
	t.Run("escaped char", func(t *testing.T) {
		_, matched, err := parse(`'\n'`)
		assert.Equal(t, &ast.BasicLit{
			Kind:  token.CHAR,
			Value: "'\\n'",
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("unicode value", func(t *testing.T) {
		_, matched, err := parse(`'c'`)
		assert.Equal(t, &ast.BasicLit{
			Kind:  token.CHAR,
			Value: "'c'",
		}, matched)
		assert.NoError(t, err)
	})
}

func TestIncDecStmt(t *testing.T) {
	parse := stringParser(IncDecStmt)
	t.Run("inc", func(t *testing.T) {
		_, matched, err := parse(`(inc i)`)
		assert.Equal(t, &ast.IncDecStmt{
			X:   ast.NewIdent("i"),
			Tok: token.INC,
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("dec", func(t *testing.T) {
		_, matched, err := parse(`(dec i)`)
		assert.Equal(t, &ast.IncDecStmt{
			X:   ast.NewIdent("i"),
			Tok: token.DEC,
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("expr", func(t *testing.T) {
		_, matched, err := parse(`(dec (intFn))`)
		assert.Equal(t, &ast.IncDecStmt{
			X: &ast.CallExpr{
				Fun: ast.NewIdent("intFn"),
			},
			Tok: token.DEC,
		}, matched)
		assert.NoError(t, err)
	})
}

func TestBlock(t *testing.T) {
	parse := stringParser(Block)
	t.Run("single expression", func(t *testing.T) {
		_, matched, err := parse(`(+ 1 2)`)
		assert.Equal(t, &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{X: &ast.BinaryExpr{
					X:  intLit(1),
					Op: token.ADD,
					Y:  intLit(2),
				}},
			},
		}, matched)
		assert.NoError(t, err)
	})
	t.Run("do expression", func(t *testing.T) {
		_, matched, err := parse(`(do (+ 1 2) (inc i))`)
		assert.Equal(t, &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{X: &ast.BinaryExpr{
					X:  intLit(1),
					Op: token.ADD,
					Y:  intLit(2),
				}},
				&ast.IncDecStmt{
					X:   ast.NewIdent("i"),
					Tok: token.INC,
				},
			},
		}, matched)
		assert.NoError(t, err)
	})
}

func TestForStmt(t *testing.T) {
	parse := stringParser(ForStmt)
	t.Run("init, cond and post", func(t *testing.T) {
		_, matched, err := parse(`(for (define i 0) (< i 10) (inc i) (println i))`)
		assert.NoError(t, err)
		assert.Equal(t, &ast.ForStmt{
			Init: &ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent("i")},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{intLit(0)},
			},
			Cond: &ast.BinaryExpr{
				X:  ast.NewIdent("i"),
				Op: token.LSS,
				Y:  intLit(10),
			},
			Post: &ast.IncDecStmt{
				X:   ast.NewIdent("i"),
				Tok: token.INC,
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{
				Fun:  ast.NewIdent("println"),
				Args: []ast.Expr{ast.NewIdent("i")},
			}}}},
		}, matched)
	})
}
