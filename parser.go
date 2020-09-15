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

func literal(s string) func(input string) (remaining string, matched interface{}, err error) {
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

func identifier(input string) (remaining string, matched interface{}, err error) {
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

func pair(p1, p2 Parser) Parser {
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

func left(p1, p2 Parser) Parser {
	p := pair(p1, p2)
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

func right(p1, p2 Parser) Parser {
	p := pair(p1, p2)
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

func oneOrMore(p Parser) Parser {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		remaining, match, err := p(remaining)
		if err != nil {
			return
		}
		matches := []interface{}{match}
		for {
			remaining, match, err = p(remaining)
			if err != nil {
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

func zeroOrMore(p Parser) Parser {
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

func anyChar(input string) (remaining string, matched interface{}, err error) {
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

func pred(p Parser, f func(matched interface{}) bool) Parser {
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

func whitespaceChar() Parser {
	return pred(anyChar, func(matched interface{}) bool {
		return unicode.IsSpace(matched.(rune))
	})
}

func space1() Parser {
	return oneOrMore(whitespaceChar())
}

func space0() Parser {
	return zeroOrMore(whitespaceChar())
}

func mapResult(p Parser, f func(matched interface{}) interface{}) Parser {
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

func quotedString() Parser {
	return mapResult(right(
		literal(`"`),
		left(
			zeroOrMore(pred(anyChar, func(matched interface{}) bool {
				return matched.(rune) != '"'
			})),
			literal(`"`))),
		func(matched interface{}) interface{} {
			var s strings.Builder
			for _, r := range matched.([]interface{}) {
				s.WriteRune(r.(rune))
			}
			return s.String()
		},
	)
}

func choice(ps ...Parser) Parser {
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

func whitespaceWrap(p Parser) Parser {
	return right(space0(), left(p, space0()))
}

func sexp() Parser {
	return right(literal("("),
		left(
			zeroOrMore(
				whitespaceWrap(choice(identifier, quotedString())),
			),
			literal(")"),
		),
	)
}
