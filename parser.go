package jo

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Source struct {
	Content string
	Offset  int
}

func (s Source) Advance(n int) Source {
	if n > len(s.Content) {
		return Source{
			Offset: s.Offset,
		}
	}
	return Source{
		Content: s.Content[n:],
		Offset:  s.Offset + n,
	}
}

func (s Source) Peek() string {
	for _, r := range s.Content {
		return string(r)
	}
	return ""
}

func NewSource(content string) Source {
	return Source{
		Content: content,
	}
}

type ParseError struct {
	Offset  int
	Message string
}

func (p *ParseError) Error() string {
	return fmt.Sprintf("offset %d: %s", p.Offset, p.Message)
}

func NewParseError(offset int, message string) error {
	return &ParseError{
		Offset:  offset,
		Message: message,
	}
}

type Parser interface {
	Parse(input Source) (remaining Source, matched interface{}, err error)
}

type ParserFunc func(input Source) (remaining Source, matched interface{}, err error)

func (p ParserFunc) Parse(input Source) (remaining Source, matched interface{}, err error) {
	return p(input)
}

func Literal(s string) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		if strings.HasPrefix(remaining.Content, s) {
			remaining = remaining.Advance(len(s))
			matched = s
			return
		}
		err = NewParseError(remaining.Offset, fmt.Sprintf("wanted a literal %q, got: %q", s, remaining.Peek()))
		return
	}
}

// Identifier matches an identifier string.
var Identifier = ParserFunc(func(input Source) (remaining Source, matched interface{}, err error) {
	remaining = input
	var match strings.Builder
	for i, r := range remaining.Content {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			err = NewParseError(remaining.Offset, fmt.Sprintf("wanted identifier, got %q", r))
			return
		}
		if !unicode.IsLetter(r) && r != '_' && !unicode.IsDigit(r) {
			break
		}
		match.WriteRune(r)
	}
	matched = match.String()
	remaining = remaining.Advance(match.Len())
	return
})

// Ident matches an &ast.Ident node.
var Ident = Map(Identifier, func(matched interface{}) interface{} {
	ident := matched.(string)
	tok := token.Lookup(ident)
	if tok == token.IDENT {
		return &ast.Ident{
			Name: matched.(string),
		}
	}
	return tok
})

var OperandName = Choice(
	QualifiedIdent,
	Ident,
)

type MatchedPair struct {
	Left  interface{}
	Right interface{}
}

func Pair(p1, p2 Parser) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		r, left, err := p1.Parse(remaining)
		if err != nil {
			return
		}
		r, right, err := p2.Parse(r)
		if err != nil {
			return
		}
		remaining = r
		matched = MatchedPair{Left: left, Right: right}
		return
	}
}

func Sequence(ps ...Parser) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		r := remaining
		var matches []interface{}
		for _, p := range ps {
			var m interface{}
			r, m, err = p.Parse(r)
			if err != nil {
				return
			}
			matches = append(matches, m)
		}
		remaining = r
		matched = matches
		return
	}
}

func Left(p1, p2 Parser) ParserFunc {
	p := Pair(p1, p2)
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		remaining, pair, err := p(remaining)
		if err != nil {
			return
		}
		matched = pair.(MatchedPair).Left
		return
	}
}

func Right(p1, p2 Parser) ParserFunc {
	p := Pair(p1, p2)
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		remaining, pair, err := p(remaining)
		if err != nil {
			return
		}
		matched = pair.(MatchedPair).Right
		return
	}
}

func OneOrMore(p Parser) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		remaining, match, err := p.Parse(remaining)
		if err != nil {
			return
		}
		matches := []interface{}{match}
		for {
			var e error
			remaining, match, e = p.Parse(remaining)
			if e != nil {
				break
			}
			matches = append(matches, match)
			if remaining.Content == "" {
				break
			}
		}
		matched = matches
		return
	}
}

func ZeroOrMore(p Parser) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		matches := make([]interface{}, 0)
		for {
			var match interface{}
			var _err error
			remaining, match, _err = p.Parse(remaining)
			if _err != nil {
				break
			}
			matches = append(matches, match)
			if remaining.Content == "" {
				break
			}
		}
		matched = matches
		return
	}
}

func Optional(p Parser) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		r, matched, e := p.Parse(remaining)
		if e != nil {
			matched = nil
			return
		}
		remaining = r
		return
	}
}

var AnyChar = ParserFunc(func(input Source) (remaining Source, matched interface{}, err error) {
	remaining = input
	r, size := utf8.DecodeRuneInString(remaining.Content)
	if r == utf8.RuneError {
		if size == 1 {
			err = NewParseError(remaining.Offset, "wanted any character, got invalid UTF-8 encoding")
		} else {
			err = NewParseError(remaining.Offset, "wanted any character, got \"\"")
		}
		return
	}
	remaining = remaining.Advance(size)
	matched = r
	return
})

func Pred(p Parser, f func(matched interface{}) bool) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		r, m, err := p.Parse(remaining)
		if err != nil {
			return
		}
		if f(m) {
			remaining = r
			matched = m
		} else {
			err = NewParseError(remaining.Offset, "predicate failed")
		}
		return
	}
}

func WhitespaceChar() ParserFunc {
	return Pred(AnyChar, func(matched interface{}) bool {
		return unicode.IsSpace(matched.(rune))
	})
}

func OneOrMoreWhitespaceChars() ParserFunc {
	return OneOrMore(WhitespaceChar())
}

func ZeroOrMoreWhitespaceChars() ParserFunc {
	return ZeroOrMore(WhitespaceChar())
}

func Map(p Parser, f func(matched interface{}) interface{}) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		remaining, matched, err = p.Parse(remaining)
		if err != nil {
			return
		}
		matched = f(matched)
		return
	}
}

func QuotedString() ParserFunc {
	return Map(Right(
		Literal(`"`),
		Left(
			ZeroOrMore(Pred(AnyChar, func(matched interface{}) bool {
				return matched.(rune) != '"'
			})),
			Literal(`"`))),
		func(matched interface{}) interface{} {
			var s strings.Builder
			for _, r := range matched.([]interface{}) {
				s.WriteRune(r.(rune))
			}
			return s.String()
		},
	)
}

func Choice(ps ...Parser) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		var r Source
		var m interface{}
		for _, p := range ps {
			r, m, err = p.Parse(remaining)
			if err == nil {
				remaining = r
				matched = m
				return
			}
		}
		return
	}
}

func WhitespaceWrap(p Parser) ParserFunc {
	return Right(ZeroOrMoreWhitespaceChars(), Left(p, ZeroOrMoreWhitespaceChars()))
}

type _decimalFloatLit struct{}

func (*_decimalFloatLit) Parse(input Source) (remaining Source, matched interface{}, err error) {
	return Map(
		Pair(OneOrMore(decimalDigit), Right(Rune('.'), OneOrMore(decimalDigit))),
		func(matched interface{}) interface{} {
			pair := matched.(MatchedPair)
			var intPart strings.Builder
			for _, v := range pair.Left.([]interface{}) {
				intPart.WriteRune(v.(rune))
			}
			var fractionalPart strings.Builder
			for _, v := range pair.Right.([]interface{}) {
				fractionalPart.WriteRune(v.(rune))
			}
			return &ast.BasicLit{
				Kind:  token.FLOAT,
				Value: intPart.String() + "." + fractionalPart.String(),
			}
		},
	)(input)
}

var decimalFloatLit *_decimalFloatLit

var decimalDigit = Pred(AnyChar, func(matched interface{}) bool {
	return unicode.IsDigit(matched.(rune))
})

func decimalLit() ParserFunc {
	nonZeroDigit := Pred(decimalDigit, func(matched interface{}) bool {
		return matched != '0'
	})
	nonZeroDecimalLit := Map(Pair(nonZeroDigit, ZeroOrMore(decimalDigit)), func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		first := pair.Left.(rune)
		rest := pair.Right.([]interface{})
		var s strings.Builder
		s.WriteRune(first)
		for _, r := range rest {
			s.WriteRune(r.(rune))
		}
		return s.String()
	})
	return Map(Choice(Literal("0"), nonZeroDecimalLit), func(matched interface{}) interface{} {
		return &ast.BasicLit{
			Kind:  token.INT,
			Value: matched.(string),
		}
	})
}

func stringLit() ParserFunc {
	return Map(QuotedString(), func(matched interface{}) interface{} {
		return &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + matched.(string) + `"`,
		}
	})
}

func basicLit() ParserFunc {
	return Choice(decimalFloatLit, decimalLit(), stringLit())
}

func Rune(r rune) ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		c, size := utf8.DecodeRuneInString(remaining.Content)
		if c == utf8.RuneError {
			if size == 1 {
				err = NewParseError(remaining.Offset, fmt.Sprintf("wanted a literal %q, got invalid UTF-8 encoding", r))
			} else {
				err = NewParseError(remaining.Offset, fmt.Sprintf("wanted a literal %q, got \"\"", r))
			}
			return
		}
		if r != c {
			err = NewParseError(remaining.Offset, fmt.Sprintf("wanted a literal %q, got %q", r, c))
			return
		}
		remaining = remaining.Advance(size)
		matched = c
		return
	}
}

func Parenthesized(p Parser) ParserFunc {
	return Right(Rune('('),
		Left(p,
			Rune(')')),
	)
}

func PackageClause() ParserFunc {
	return Parenthesized(Right(Literal(token.PACKAGE.String()), Right(OneOrMoreWhitespaceChars(), Ident)))
}

func MapConst(p Parser, v interface{}) Parser {
	return Map(p, func(interface{}) interface{} {
		return v
	})
}

var BinaryOp = Choice(
	MapConst(Rune('+'), token.ADD),
	MapConst(Rune('*'), token.MUL),
	MapConst(Rune('/'), token.QUO),
	MapConst(Rune('='), token.EQL),
	MapConst(Rune('<'), token.LSS),
	MapConst(Rune('>'), token.GTR),
)

type binaryExpr struct{}

func (*binaryExpr) Parse(input Source) (remaining Source, matched interface{}, err error) {
	return Map(
		Parenthesized(Pair(BinaryOp, Right(OneOrMoreWhitespaceChars(), Pair(Expr, Right(OneOrMoreWhitespaceChars(), Expr))))),
		func(matched interface{}) interface{} {
			pair := matched.(MatchedPair)
			operands := pair.Right.(MatchedPair)
			return &ast.BinaryExpr{
				X:  operands.Left.(ast.Expr),
				Op: pair.Left.(token.Token),
				Y:  operands.Right.(ast.Expr),
			}
		},
	)(input)
}

var BinaryExpr *binaryExpr

type callExpr struct{}

func (*callExpr) Parse(input Source) (remaining Source, matched interface{}, err error) {
	return Map(Parenthesized(Pair(OperandName, ZeroOrMore(Right(OneOrMoreWhitespaceChars(), Expr)))),
		func(matched interface{}) interface{} {
			pair := matched.(MatchedPair)
			fun := pair.Left.(ast.Expr)
			var args []ast.Expr
			for _, basicLit := range pair.Right.([]interface{}) {
				args = append(args, basicLit.(ast.Expr))
			}
			return &ast.CallExpr{
				Fun:  fun,
				Args: args,
			}
		})(input)
}

var CallExpr *callExpr

type expr struct{}

func (*expr) Parse(input Source) (remaining Source, matched interface{}, err error) {
	return Choice(basicLit(), BinaryExpr, Selector, CallExpr, OperandName)(input)
}

var Expr *expr

type selectorCall struct {
	Sel  *ast.Ident
	Args []ast.Expr
}

var SelectorCall = Map(Parenthesized(Pair(Ident, ZeroOrMore(Right(OneOrMoreWhitespaceChars(), Expr)))), func(matched interface{}) interface{} {
	match := matched.(MatchedPair)
	var args []ast.Expr
	for _, e := range match.Right.([]interface{}) {
		args = append(args, e.(ast.Expr))
	}
	return selectorCall{
		Sel:  match.Left.(*ast.Ident),
		Args: args,
	}
})

type selector struct{}

func (*selector) Parse(input Source) (remaining Source, matched interface{}, err error) {
	return Map(
		Parenthesized(Right(
			Literal("sel"),
			Right(OneOrMoreWhitespaceChars(), Pair(
				Expr,
				OneOrMore(Right(OneOrMoreWhitespaceChars(), Choice(SelectorCall, Ident))))))),
		func(matched interface{}) interface{} {
			pair := matched.(MatchedPair)
			expr := pair.Left.(ast.Expr)
			for _, p := range pair.Right.([]interface{}) {
				switch x := p.(type) {
				case *ast.Ident:
					expr = &ast.SelectorExpr{
						X:   expr,
						Sel: x,
					}
				case selectorCall:
					expr = &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   expr,
							Sel: x.Sel,
						},
						Args: x.Args,
					}
				}
			}
			return expr
		})(input)
}

var Selector *selector

type structType struct{}

func (*structType) Parse(input Source) (remaining Source, matched interface{}, err error) {
	return Map(Parenthesized(
		Right(
			Literal(token.STRUCT.String()),
			ZeroOrMore(
				Right(
					ZeroOrMoreWhitespaceChars(),
					Parenthesized(
						Pair(Ident, Right(OneOrMoreWhitespaceChars(), Ident))))))),
		func(matched interface{}) interface{} {
			matches := matched.([]interface{})
			var fields []*ast.Field
			for _, m := range matches {
				pair := m.(MatchedPair)
				fields = append(fields, &ast.Field{
					Names: []*ast.Ident{pair.Left.(*ast.Ident)},
					Type:  pair.Right.(*ast.Ident),
				})
			}
			return &ast.StructType{
				Fields: &ast.FieldList{
					List: fields,
				},
			}
		},
	)(input)
}

var StructType *structType

type typeDecl struct{}

func (*typeDecl) Parse(input Source) (remaining Source, matched interface{}, err error) {
	return Map(Parenthesized(Right(
		Literal(token.TYPE.String()), Right(OneOrMoreWhitespaceChars(),
			Pair(Ident, Right(OneOrMoreWhitespaceChars(),
				StructType))))),
		func(matched interface{}) interface{} {
			pair := matched.(MatchedPair)
			return &ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.TypeSpec{
						Name: pair.Left.(*ast.Ident),
						Type: pair.Right.(*ast.StructType),
					},
				},
			}
		})(input)
}

var TypeDecl *typeDecl

type statementList struct{}

func (*statementList) Parse(input Source) (remaining Source, matched interface{}, err error) {
	return Map(
		ZeroOrMore(WhitespaceWrap(Statement)),
		func(matched interface{}) interface{} {
			var stmts []ast.Stmt
			for _, m := range matched.([]interface{}) {
				switch v := m.(type) {
				case ast.Expr:
					stmts = append(stmts, &ast.ExprStmt{X: v})
				case ast.Stmt:
					stmts = append(stmts, v)
				}
			}
			return stmts
		},
	)(input)
}

var StatementList *statementList

var Statement = Choice(IfStmt, CallExpr)

var DoExpr = Map(Parenthesized(Right(
	Literal("do"),
	Optional(Right(OneOrMoreWhitespaceChars(),
		StatementList)))),
	func(matched interface{}) interface{} {
		if matched == nil {
			return &ast.BlockStmt{
				List: []ast.Stmt{},
			}
		}
		return nil
	},
)

func Noop() ParserFunc {
	return func(input Source) (remaining Source, matched interface{}, err error) {
		remaining = input
		return
	}
}

var FunctionDecl = Map(Parenthesized(Right(
	Literal(token.FUNC.String()), Right(OneOrMoreWhitespaceChars(), Pair(
		Ident, Right(Right(OneOrMoreWhitespaceChars(), Parenthesized(Noop())), WhitespaceWrap(
			StatementList)))))),
	func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		return &ast.FuncDecl{
			Name: pair.Left.(*ast.Ident),
			Type: &ast.FuncType{Params: &ast.FieldList{}},
			Body: &ast.BlockStmt{
				List: pair.Right.([]ast.Stmt),
			},
		}
	},
)

var TopLevelDecl = Choice(TypeDecl, FunctionDecl)

var ImportDecl = Map(
	Parenthesized(Right(Literal(token.IMPORT.String()), OneOrMore(Right(OneOrMoreWhitespaceChars(), stringLit())))),
	func(matched interface{}) interface{} {
		matches := matched.([]interface{})
		var specs []ast.Spec
		for _, path := range matches {
			specs = append(specs, &ast.ImportSpec{
				Path: path.(*ast.BasicLit),
			})
		}
		return &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: specs,
		}
	})

var QualifiedIdent = Map(
	Pair(Ident, Right(Rune('.'), Ident)),
	func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		x := pair.Left.(*ast.Ident)
		sel := pair.Right.(*ast.Ident)
		return &ast.SelectorExpr{
			X:   x,
			Sel: sel,
		}
	})

var IfStmt = Map(Parenthesized(Right(
	Literal(token.IF.String()), Pair(Right(ZeroOrMoreWhitespaceChars(),
		Expr), Right(ZeroOrMoreWhitespaceChars(),
		Choice(DoExpr, Expr))))),
	func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		cond := pair.Left.(ast.Expr)
		var body *ast.BlockStmt
		switch v := pair.Right.(type) {
		case *ast.BlockStmt:
			body = v
		case ast.Expr:
			body = &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: v},
				},
			}
		}
		return &ast.IfStmt{
			Cond: cond,
			Body: body,
		}
	})

var SourceFile = Map(
	Sequence(
		WhitespaceWrap(PackageClause()),
		WhitespaceWrap(ZeroOrMore(WhitespaceWrap(ImportDecl))),
		WhitespaceWrap(OneOrMore(WhitespaceWrap(TopLevelDecl)))),
	func(matched interface{}) interface{} {
		matches := matched.([]interface{})
		pkgName := matches[0].(*ast.Ident)
		var decls []ast.Decl
		for _, d := range matches[1].([]interface{}) {
			decls = append(decls, d.(ast.Decl))
		}
		for _, d := range matches[2].([]interface{}) {
			decls = append(decls, d.(ast.Decl))
		}
		return &ast.File{
			Name:  pkgName,
			Decls: decls,
		}
	})
