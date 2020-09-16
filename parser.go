package jo

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Symbol, string

type Parser func(input string) (remaining string, matched interface{}, err error)

func Literal(s string) func(input string) (remaining string, matched interface{}, err error) {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		if strings.HasPrefix(remaining, s) {
			remaining = strings.TrimPrefix(remaining, s)
			matched = s
			return
		}
		err = errors.New(fmt.Sprintf("wanted a literal %q, got: %q", s, remaining))
		return
	}
}

func Identifier(input string) (remaining string, matched interface{}, err error) {
	remaining = input
	var match strings.Builder
	for i, r := range remaining {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			err = errors.New(remaining)
			return
		}
		if !unicode.IsLetter(r) && r != '_' && !unicode.IsDigit(r) {
			break
		}
		match.WriteRune(r)
	}
	matched = match.String()
	remaining = remaining[match.Len():]
	return
}

var unqualifiedIdent = Map(Identifier, func(matched interface{}) interface{} {
	ident := matched.(string)
	tok := token.Lookup(ident)
	if tok == token.IDENT {
		return &ast.Ident{
			Name: matched.(string),
		}
	}
	return tok
})

var identifier = Choice(
	QualifiedIdent,
	unqualifiedIdent,
)

type MatchedPair struct {
	Left  interface{}
	Right interface{}
}

func Pair(p1, p2 Parser) Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		r, left, err := p1(remaining)
		if err != nil {
			return
		}
		r, right, err := p2(r)
		if err != nil {
			return
		}
		remaining = r
		matched = MatchedPair{Left: left, Right: right}
		return
	}
}

func List(ps ...Parser) Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		r := remaining
		var matches []interface{}
		for _, p := range ps {
			var m interface{}
			r, m, err = p(r)
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

func Left(p1, p2 Parser) Parser {
	p := Pair(p1, p2)
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		remaining, pair, err := p(remaining)
		if err != nil {
			return
		}
		matched = pair.(MatchedPair).Left
		return
	}
}

func Right(p1, p2 Parser) Parser {
	p := Pair(p1, p2)
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		remaining, pair, err := p(remaining)
		if err != nil {
			return
		}
		matched = pair.(MatchedPair).Right
		return
	}
}

func OneOrMore(p Parser) Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		remaining, match, err := p(remaining)
		if err != nil {
			return
		}
		matches := []interface{}{match}
		for {
			var e error
			remaining, match, e = p(remaining)
			if e != nil {
				break
			}
			matches = append(matches, match)
			if remaining == "" {
				break
			}
		}
		matched = matches
		return
	}
}

func ZeroOrMore(p Parser) Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		matches := make([]interface{}, 0)
		for {
			var match interface{}
			var _err error
			remaining, match, _err = p(remaining)
			if _err != nil {
				break
			}
			matches = append(matches, match)
			if remaining == "" {
				break
			}
		}
		matched = matches
		return
	}
}

func AnyChar(input string) (remaining string, matched interface{}, err error) {
	remaining = input
	r, size := utf8.DecodeRuneInString(remaining)
	if r == utf8.RuneError {
		err = errors.New(remaining)
		return
	}
	remaining = input[size:]
	matched = r
	return
}

func Pred(p Parser, f func(matched interface{}) bool) Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		r, m, err := p(remaining)
		if err != nil {
			return
		}
		if f(m) {
			remaining = r
			matched = m
		} else {
			err = errors.New(input)
		}
		return
	}
}

func WhitespaceChar() Parser {
	return Pred(AnyChar, func(matched interface{}) bool {
		return unicode.IsSpace(matched.(rune))
	})
}

func OneOrMoreWhitespaceChars() Parser {
	return OneOrMore(WhitespaceChar())
}

func ZeroOrMoreWhitespaceChars() Parser {
	return ZeroOrMore(WhitespaceChar())
}

func Map(p Parser, f func(matched interface{}) interface{}) Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		remaining, matched, err = p(remaining)
		if err != nil {
			return
		}
		matched = f(matched)
		return
	}
}

func QuotedString() Parser {
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

func Choice(ps ...Parser) Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		var r string
		var m interface{}
		for _, p := range ps {
			r, m, err = p(remaining)
			if err == nil {
				remaining = r
				matched = m
				return
			}
		}
		return
	}
}

func WhitespaceWrap(p Parser) Parser {
	return Right(ZeroOrMoreWhitespaceChars(), Left(p, ZeroOrMoreWhitespaceChars()))
}

func decimalLit() Parser {
	decimalDigit := Pred(AnyChar, func(matched interface{}) bool {
		return unicode.IsDigit(matched.(rune))
	})
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

func stringLit() Parser {
	return Map(QuotedString(), func(matched interface{}) interface{} {
		return &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + matched.(string) + `"`,
		}
	})
}

func basicLit() Parser {
	return Choice(decimalLit(), stringLit())
}

func Rune(r rune) Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		c, size := utf8.DecodeRuneInString(remaining)
		if c == utf8.RuneError {
			err = errors.New(remaining)
			return
		}
		if r != c {
			err = errors.New(fmt.Sprintf("wanted a literal %q, got %q", r, c))
			return
		}
		remaining = input[size:]
		matched = c
		return
	}
}

func SExpr(p Parser) Parser {
	return Right(Rune('('),
		Left(p,
			Rune(')')),
	)
}

func PackageClause() Parser {
	return SExpr(Right(Literal("package"), Right(OneOrMoreWhitespaceChars(), identifier)))
}

func CallExpr() Parser {
	return Map(SExpr(Pair(identifier, Right(OneOrMoreWhitespaceChars(), ZeroOrMore(WhitespaceWrap(basicLit()))))),
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
		})
}

func StatementList() Parser {
	return ZeroOrMore(WhitespaceWrap(CallExpr()))
}

func Noop() Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		return
	}
}

func FunctionDecl() Parser {
	return Map(SExpr(Right(Literal("func"), Right(OneOrMoreWhitespaceChars(), Pair(identifier, Right(Right(OneOrMoreWhitespaceChars(), SExpr(Noop())), WhitespaceWrap(StatementList())))))),
		func(matched interface{}) interface{} {
			pair := matched.(MatchedPair)
			name := pair.Left.(*ast.Ident)
			var body []ast.Stmt
			for _, callExpr := range pair.Right.([]interface{}) {
				body = append(body, &ast.ExprStmt{X: callExpr.(*ast.CallExpr)})
			}
			return &ast.FuncDecl{
				Name: name,
				Type: &ast.FuncType{Params: &ast.FieldList{}},
				Body: &ast.BlockStmt{
					List: body,
				},
			}
		},
	)
}

func ImportDecl() Parser {
	return Map(
		SExpr(Right(Literal("import"), OneOrMore(Right(OneOrMoreWhitespaceChars(), stringLit())))),
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
}

var QualifiedIdent = Map(
	Pair(unqualifiedIdent, Right(Rune('.'), unqualifiedIdent)),
	func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		x := pair.Left.(*ast.Ident)
		sel := pair.Right.(*ast.Ident)
		return &ast.SelectorExpr{
			X:   x,
			Sel: sel,
		}
	})

var SourceFile = Map(
	List(
		WhitespaceWrap(PackageClause()),
		WhitespaceWrap(ZeroOrMore(WhitespaceWrap(ImportDecl()))),
		WhitespaceWrap(OneOrMore(WhitespaceWrap(FunctionDecl())))),
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
