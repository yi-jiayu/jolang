package jo

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Symbol, string

type Parser func(input string) (remaining string, matched []string, err error)

func literal(s string) func(input string) (remaining string, matched []string, err error) {
	return func(input string) (remaining string, matched []string, err error) {
		remaining = input
		if strings.HasPrefix(input, s) {
			remaining = strings.TrimPrefix(input, s)
			matched = []string{s}
			return
		}
		err = errors.New(input)
		return
	}
}

func identifier(input string) (remaining string, matched []string, err error) {
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
	matched = []string{match.String()}
	remaining = remaining[match.Len():]
	return
}

func pair(p1, p2 Parser) Parser {
	return func(input string) (remaining string, matched []string, err error) {
		remaining = input
		remaining, matched1, err := p1(remaining)
		if err != nil {
			return
		}
		remaining, matched2, err := p2(remaining)
		if err != nil {
			return
		}
		matched = append(matched1, matched2...)
		return
	}
}

func head(p1, p2 Parser) Parser {
	p := pair(p1, p2)
	return func(input string) (remaining string, matched []string, err error) {
		remaining = input
		remaining, match, err := p(remaining)
		if err != nil {
			return
		}
		matched = match[:1]
		return
	}
}

func tail(p1, p2 Parser) Parser {
	p := pair(p1, p2)
	return func(input string) (remaining string, matched []string, err error) {
		remaining = input
		remaining, match, err := p(remaining)
		if err != nil {
			return
		}
		matched = match[1:]
		return
	}
}

func oneOrMore(p Parser) Parser {
	return func(input string) (remaining string, matched []string, err error) {
		remaining = input
		remaining, match, err := p(remaining)
		if err != nil {
			return
		}
		matched = append(matched, match...)
		for {
			remaining, match, err = p(remaining)
			if err != nil {
				break
			}
			matched = append(matched, match...)
			if remaining == "" {
				break
			}
		}
		return
	}
}

func zeroOrMore(p Parser) Parser {
	return func(input string) (remaining string, matched []string, err error) {
		remaining = input
		for {
			var match []string
			var _err error
			remaining, match, _err = p(remaining)
			if _err != nil {
				break
			}
			matched = append(matched, match...)
			if remaining == "" {
				break
			}
		}
		return
	}
}

func anyChar(input string) (remaining string, matched []string, err error) {
	r, size := utf8.DecodeRuneInString(input)
	if r == utf8.RuneError {
		err = errors.New(remaining)
		return
	}
	remaining = input[size:]
	matched = []string{string(r)}
	return
}

func pred(p Parser, f func(matched []string) bool) Parser {
	return func(input string) (remaining string, matched []string, err error) {
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
