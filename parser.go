package jo

import (
	"errors"
	"fmt"
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

type SexpParser struct{}

func (p SexpParser) Parse(input string) (remaining string, matched interface{}, err error) {
	return Right(Literal("("),
		Left(
			ZeroOrMore(
				WhitespaceWrap(Choice(Identifier, QuotedString(), p.Parse)),
			),
			Literal(")"),
		),
	)(input)
}

func SExpr() Parser {
	return SexpParser{}.Parse
}

func SExprs() Parser {
	return ZeroOrMore(WhitespaceWrap(SExpr()))
}

func PackageClause() Parser {
	return Right(Literal("package"), Right(OneOrMoreWhitespaceChars(), Identifier))
}
