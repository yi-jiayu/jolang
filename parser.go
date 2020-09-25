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
	Content *string
	Offset  int
}

func (s Source) Remaining() string {
	return (*s.Content)[s.Offset:]
}

// Finished indicates whether the input has been fully consumed.
func (s Source) Finished() bool {
	return s.Offset >= len(*s.Content)
}

func (s Source) Advance(n int) Source {
	if n+s.Offset >= len(*s.Content) {
		return Source{
			Content: s.Content,
			Offset:  len(*s.Content),
		}
	}
	return Source{
		Content: s.Content,
		Offset:  s.Offset + n,
	}
}

// PeekRune calls utf8.DecodeRuneInString on the unparsed input.
func (s Source) PeekRune() (rune, int) {
	return utf8.DecodeRuneInString(s.Remaining())
}

func (s Source) Peek() string {
	for _, r := range *s.Content {
		return string(r)
	}
	return ""
}

func NewSource(content string) Source {
	return Source{
		Content: &content,
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
	Parse(input Source) (output Source, matched interface{}, err error)
}

type ParserFunc func(input Source) (output Source, matched interface{}, err error)

func (p ParserFunc) Parse(input Source) (output Source, matched interface{}, err error) {
	return p(input)
}

func Literal(s string) ParserFunc {
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		if strings.HasPrefix(output.Remaining(), s) {
			output = output.Advance(len(s))
			matched = s
			return
		}
		err = NewParseError(output.Offset, fmt.Sprintf("wanted a literal %q, got: %q", s, output.Peek()))
		return
	}
}

// Identifier matches an identifier string.
var Identifier = ParserFunc(func(input Source) (output Source, matched interface{}, err error) {
	output = input
	var match strings.Builder
	for i, r := range output.Remaining() {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			err = NewParseError(output.Offset, fmt.Sprintf("wanted identifier, got %q", r))
			return
		}
		if !unicode.IsLetter(r) && r != '_' && !unicode.IsDigit(r) {
			break
		}
		match.WriteRune(r)
	}
	matched = match.String()
	output = output.Advance(match.Len())
	return
})

var Ident = Map(Identifier, func(matched interface{}) interface{} {
	return ast.NewIdent(matched.(string))
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
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		r, left, err := p1.Parse(output)
		if err != nil {
			return
		}
		r, right, err := p2.Parse(r)
		if err != nil {
			return
		}
		output = r
		matched = MatchedPair{Left: left, Right: right}
		return
	}
}

func Sequence(ps ...Parser) ParserFunc {
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		r := output
		var matches []interface{}
		for _, p := range ps {
			var m interface{}
			r, m, err = p.Parse(r)
			if err != nil {
				return
			}
			matches = append(matches, m)
		}
		output = r
		matched = matches
		return
	}
}

func Left(p1, p2 Parser) ParserFunc {
	p := Pair(p1, p2)
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		output, pair, err := p(output)
		if err != nil {
			return
		}
		matched = pair.(MatchedPair).Left
		return
	}
}

func Right(p1, p2 Parser) ParserFunc {
	p := Pair(p1, p2)
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		output, pair, err := p(output)
		if err != nil {
			return
		}
		matched = pair.(MatchedPair).Right
		return
	}
}

func OneOrMore(p Parser) ParserFunc {
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		output, match, err := p.Parse(output)
		if err != nil {
			return
		}
		matches := []interface{}{match}
		for {
			if output.Finished() {
				break
			}
			var e error
			output, match, e = p.Parse(output)
			if e != nil {
				break
			}
			matches = append(matches, match)
		}
		matched = matches
		return
	}
}

func ZeroOrMore(p Parser) ParserFunc {
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		matches := make([]interface{}, 0)
		for {
			if output.Finished() {
				break
			}
			var match interface{}
			var _err error
			output, match, _err = p.Parse(output)
			if _err != nil {
				break
			}
			matches = append(matches, match)
		}
		matched = matches
		return
	}
}

//goland:noinspection GoUnusedExportedFunction
func Optional(p Parser) ParserFunc {
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		r, matched, e := p.Parse(output)
		if e != nil {
			matched = nil
			return
		}
		output = r
		return
	}
}

var AnyChar = ParserFunc(func(input Source) (output Source, matched interface{}, err error) {
	output = input
	r, size := output.PeekRune()
	if r == utf8.RuneError {
		if size == 1 {
			err = NewParseError(output.Offset, "wanted any character, got invalid UTF-8 encoding")
		} else {
			err = NewParseError(output.Offset, "wanted any character, got \"\"")
		}
		return
	}
	output = output.Advance(size)
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
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		output, matched, err = p.Parse(output)
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
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		var r Source
		var m interface{}
		for _, p := range ps {
			r, m, err = p.Parse(output)
			if err == nil {
				output = r
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

func (*_decimalFloatLit) Parse(input Source) (output Source, matched interface{}, err error) {
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

var escapedChar = Choice(
	Literal(`\a`),
	Literal(`\b`),
	Literal(`\f`),
	Literal(`\n`),
	Literal(`\r`),
	Literal(`\t`),
	Literal(`\v`),
	Literal(`\\`),
	Literal(`\'`),
	Literal(`\"`),
)

var RuneLit = Map(
	Right(Rune('\''), Left(Choice(escapedChar, AnyChar), Rune('\''))),
	func(matched interface{}) interface{} {
		var value string
		switch v := matched.(type) {
		case string:
			value = v
		case rune:
			value = string(v)
		}
		return &ast.BasicLit{
			Kind:  token.CHAR,
			Value: `'` + value + `'`,
		}
	})

func stringLit() ParserFunc {
	return Map(QuotedString(), func(matched interface{}) interface{} {
		return &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + matched.(string) + `"`,
		}
	})
}

func basicLit() ParserFunc {
	return Choice(decimalFloatLit, decimalLit(), RuneLit, stringLit())
}

func Rune(r rune) ParserFunc {
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
		c, size := output.PeekRune()
		if c == utf8.RuneError {
			if size == 1 {
				err = NewParseError(output.Offset, fmt.Sprintf("wanted a literal %q, got invalid UTF-8 encoding", r))
			} else {
				err = NewParseError(output.Offset, fmt.Sprintf("wanted a literal %q, got \"\"", r))
			}
			return
		}
		if r != c {
			err = NewParseError(output.Offset, fmt.Sprintf("wanted a literal %q, got %q", r, c))
			return
		}
		output = output.Advance(size)
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
	MapConst(Rune('%'), token.REM),
	MapConst(Literal("!="), token.NEQ),
)

type binaryExpr struct{}

func (*binaryExpr) Parse(input Source) (output Source, matched interface{}, err error) {
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

var UnaryOp = Choice(
	MapConst(Rune('&'), token.AND),
)

var UnaryExpr = Map(Pair(UnaryOp, Expr), func(matched interface{}) interface{} {
	pair := matched.(MatchedPair)
	return &ast.UnaryExpr{
		Op: pair.Left.(token.Token),
		X:  pair.Right.(ast.Expr),
	}
})

type callExpr struct{}

func (*callExpr) Parse(input Source) (output Source, matched interface{}, err error) {
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

func (*expr) Parse(input Source) (output Source, matched interface{}, err error) {
	return Choice(basicLit(), BinaryExpr, UnaryExpr, Selector, CallExpr, OperandName)(input)
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

func (*selector) Parse(input Source) (output Source, matched interface{}, err error) {
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

func (*structType) Parse(input Source) (output Source, matched interface{}, err error) {
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

func (*typeDecl) Parse(input Source) (output Source, matched interface{}, err error) {
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

func (*statementList) Parse(input Source) (output Source, matched interface{}, err error) {
	return Map(
		ZeroOrMore(WhitespaceWrap(Statement)),
		func(matched interface{}) interface{} {
			matches := matched.([]interface{})
			stmts := make([]ast.Stmt, len(matches))
			for i, m := range matches {
				switch v := m.(type) {
				case ast.Expr:
					stmts[i] = &ast.ExprStmt{X: v}
				case ast.Stmt:
					stmts[i] = v
				}
			}
			return stmts
		},
	)(input)
}

var StatementList *statementList

var Statement = Choice(ExprSwitchStmt, ForStmt, DeclStmt, IfStmt, SimpleStmt)

var SimpleStmt = Choice(Define, Assignment, IncDecStmt, ExprStmt)

var ExprStmt = Map(Expr, func(matched interface{}) interface{} {
	return &ast.ExprStmt{X: matched.(ast.Expr)}
})

var IncDecStmt = Map(
	Parenthesized(Pair(Choice(MapConst(Keyword("inc"), token.INC), MapConst(Keyword("dec"), token.DEC)), WhitespaceWrap(Expr))),
	func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		return &ast.IncDecStmt{
			X:   pair.Right.(ast.Expr),
			Tok: pair.Left.(token.Token),
		}
	})

// DoExpr matches an S-expression starting with a "do" keyword and a StatementList, returning a slice of ast.Stmt.
var DoExpr = Map(Parenthesized(Right(
	Literal("do"),
	StatementList)),
	func(matched interface{}) interface{} {
		return matched.([]ast.Stmt)
	},
)

func Noop() ParserFunc {
	return func(input Source) (output Source, matched interface{}, err error) {
		output = input
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

type block struct{}

func (*block) Parse(input Source) (output Source, matched interface{}, err error) {
	return Map(
		Choice(DoExpr, Statement),
		func(matched interface{}) interface{} {
			switch v := matched.(type) {
			case []ast.Stmt:
				return &ast.BlockStmt{
					List: v,
				}
			case ast.Stmt:
				return &ast.BlockStmt{
					List: []ast.Stmt{v},
				}
			}
			return nil
		})(input)
}

// Block matches either a do expression or a single statement and returns a pointer to an ast.BlockStmt.
var Block *block

var IfStmt = Map(Parenthesized(Right(
	Keyword(token.IF.String()), Pair(Right(ZeroOrMoreWhitespaceChars(),
		Expr), Pair(
		WhitespaceWrap(Block),
		Optional(WhitespaceWrap(Block)))))),
	func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		cond, arms := pair.Left.(ast.Expr), pair.Right.(MatchedPair)
		var Else ast.Stmt
		if e, ok := arms.Right.(*ast.BlockStmt); ok {
			Else = e
		}
		return &ast.IfStmt{
			Cond: cond,
			Body: arms.Left.(*ast.BlockStmt),
			Else: Else,
		}
	})

var IdentifierList = Map(Choice(Parenthesized(OneOrMore(WhitespaceWrap(Ident))), Ident), func(matched interface{}) interface{} {
	switch v := matched.(type) {
	case []interface{}:
		exprs := make([]ast.Expr, len(v))
		for i, match := range v {
			exprs[i] = match.(ast.Expr)
		}
		return exprs
	case *ast.Ident:
		return []ast.Expr{v}
	}
	return nil
})

// ExpressionList matches a list of parenthesized expressions, a single identifier or a single basic literal and returns a slice of ast.Expr.
var ExpressionList = Map(Choice(Parenthesized(OneOrMore(WhitespaceWrap(Expr))), Ident, basicLit()), func(matched interface{}) interface{} {
	switch v := matched.(type) {
	case []interface{}:
		exprs := make([]ast.Expr, len(v))
		for i, match := range v {
			exprs[i] = match.(ast.Expr)
		}
		return exprs
	case *ast.Ident:
		return []ast.Expr{v}
	case *ast.BasicLit:
		return []ast.Expr{v}
	}
	return nil
})

var Define = Map(Parenthesized(Right(
	Literal("define"), Pair(Right(OneOrMoreWhitespaceChars(),
		IdentifierList), Right(OneOrMoreWhitespaceChars(),
		ExpressionList)))),
	func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		return &ast.AssignStmt{
			Lhs: pair.Left.([]ast.Expr),
			Tok: token.DEFINE,
			Rhs: pair.Right.([]ast.Expr),
		}
	},
)

func Keyword(k string) Parser {
	return Pred(Identifier, func(matched interface{}) bool {
		return matched.(string) == k
	})
}

var DeclStmt = Map(
	Parenthesized(Right(Keyword("var"), Pair(WhitespaceWrap(Ident), WhitespaceWrap(Ident)))),
	func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		return &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{pair.Left.(*ast.Ident)},
						Type:  pair.Right.(*ast.Ident),
					},
				},
			},
		}
	})

var ForStmt = Map(
	Parenthesized(Right(Keyword("for"), Sequence(
		WhitespaceWrap(SimpleStmt),
		WhitespaceWrap(Expr),
		WhitespaceWrap(SimpleStmt),
		WhitespaceWrap(Block)))),
	func(matched interface{}) interface{} {
		seq := matched.([]interface{})
		return &ast.ForStmt{
			Init: seq[0].(ast.Stmt),
			Cond: seq[1].(ast.Expr),
			Post: seq[2].(ast.Stmt),
			Body: seq[3].(*ast.BlockStmt),
		}
	})

var Assignment = Map(
	Parenthesized(Right(
		Keyword("assign"), Pair(WhitespaceWrap(
			IdentifierList), WhitespaceWrap(
			ExpressionList)))),
	func(matched interface{}) interface{} {
		pair := matched.(MatchedPair)
		return &ast.AssignStmt{
			Lhs: pair.Left.([]ast.Expr),
			Tok: token.ASSIGN,
			Rhs: pair.Right.([]ast.Expr),
		}
	})

var ExprSwitchStmt = Map(
	Parenthesized(Right(Keyword("switch"), ZeroOrMore(WhitespaceWrap(
		Choice(
			Parenthesized(Right(Keyword("case"), Pair(WhitespaceWrap(ExpressionList), WhitespaceWrap(Block)))),
			Parenthesized(Right(Keyword("default"), WhitespaceWrap(Block)))))))),
	func(matched interface{}) interface{} {
		var clauses []ast.Stmt
		for _, match := range matched.([]interface{}) {
			switch v := match.(type) {
			case *ast.BlockStmt:
				clauses = append(clauses, &ast.CaseClause{
					Body: v.List,
				})
			case MatchedPair:
				clauses = append(clauses, &ast.CaseClause{
					List: v.Left.([]ast.Expr),
					Body: v.Right.(*ast.BlockStmt).List,
				})
			}
		}
		return &ast.SwitchStmt{
			Body: &ast.BlockStmt{List: clauses},
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
