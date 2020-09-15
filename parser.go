package jo

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Symbol, string

type Parser func(input string) (remaining string, matched interface{}, err error)

func literal(s string) func(input string) (remaining string, matched interface{}, err error) {
	return func(input string) (remaining string, matched interface{}, err error) {
		remaining = input
		if strings.HasPrefix(input, s) {
			remaining = strings.TrimPrefix(input, s)
			matched = s
			return
		}
		err = errors.New(input)
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
		remaining, left, err := p1(remaining)
		if err != nil {
			return
		}
		remaining, right, err := p2(remaining)
		if err != nil {
			return
		}
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
		var matches []interface{}
		matches = append(matches, match)
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
		var matches []interface{}
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
	r, size := utf8.DecodeRuneInString(input)
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
		r, m, err := p(input)
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
