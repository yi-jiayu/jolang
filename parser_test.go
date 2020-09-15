package jo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_literal(t *testing.T) {
	parseJoe := literal("Hello Joe!")
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
		assert.EqualError(t, err, "Hello Mike!")
	}
}

func Test_identifier(t *testing.T) {
	{
		remaining, matched, err := identifier("i_am_an_identifier")
		assert.Empty(t, remaining)
		assert.Equal(t, "i_am_an_identifier", matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := identifier("not entirely an identifier")
		assert.Equal(t, " entirely an identifier", remaining)
		assert.Equal(t, "not", matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := identifier("!not at all an identifier")
		assert.EqualError(t, err, "!not at all an identifier")
	}
}

func Test_pair(t *testing.T) {
	tagOpener := pair(literal("<"), identifier)
	{
		remaining, matched, err := tagOpener("<element/>")
		assert.Equal(t, "/>", remaining)
		assert.Equal(t, MatchedPair{Left: "<", Right: "element"}, matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := tagOpener("oops")
		assert.EqualError(t, err, "oops")
	}
	{
		_, _, err := tagOpener("<!oops")
		assert.EqualError(t, err, "!oops")
	}
}

func Test_right(t *testing.T) {
	tagOpener := right(literal("<"), identifier)
	{
		remaining, matched, err := tagOpener("<element/>")
		assert.Equal(t, "/>", remaining)
		assert.Equal(t, "element", matched)
		assert.NoError(t, err)
	}
}

func Test_oneOrMore(t *testing.T) {
	p := oneOrMore(literal("ha"))
	{
		remaining, matched, err := p("hahaha")
		assert.Empty(t, remaining)
		assert.Equal(t, []interface{}{"ha", "ha", "ha"}, matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := p("ahah")
		assert.EqualError(t, err, "ahah")
	}
	{
		_, _, err := p("")
		assert.EqualError(t, err, "")
	}
}

func Test_zeroOrMore(t *testing.T) {
	p := zeroOrMore(literal("ha"))
	{
		remaining, matched, err := p("hahaha")
		assert.Empty(t, remaining)
		assert.Equal(t, []interface{}{"ha", "ha", "ha"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("ahah")
		assert.Equal(t, remaining, "ahah")
		assert.Empty(t, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p("")
		assert.Equal(t, remaining, "")
		assert.Empty(t, matched)
		assert.NoError(t, err)
	}
}

func Test_pred(t *testing.T) {
	p := pred(anyChar, func(matched interface{}) bool {
		return matched == 'o'
	})
	{
		remaining, matched, err := p("omg")
		assert.Equal(t, "mg", remaining)
		assert.Equal(t, 'o', matched)
		assert.NoError(t, err)
	}
	{
		_, _, err := p("lol")
		assert.EqualError(t, err, "lol")
	}
}
