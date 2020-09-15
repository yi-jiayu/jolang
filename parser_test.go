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
		assert.EqualError(t, err, "wanted a literal \"Hello Joe!\", got: \"Hello Mike!\"")
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
		assert.EqualError(t, err, "wanted a literal \"ha\", got: \"ahah\"")
	}
	{
		_, _, err := p("")
		assert.EqualError(t, err, "wanted a literal \"ha\", got: \"\"")
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
		remaining, _, err := p("lol")
		assert.Equal(t, "lol", remaining)
		assert.EqualError(t, err, "lol")
	}
}

func Test_quotedString(t *testing.T) {
	p := quotedString()
	remaining, matched, err := p(`"Hello Joe!"`)
	assert.Equal(t, "", remaining)
	assert.Equal(t, "Hello Joe!", matched)
	assert.NoError(t, err)
}

func Test_choice(t *testing.T) {
	p := choice(literal("package"), literal("func"))
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

func Test_sexp(t *testing.T) {
	p := sexp()
	{
		remaining, matched, err := p(`()`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p(`(import main)`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{"import", "main"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p(`(println "Hello, World")`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{"println", "Hello, World"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, matched, err := p(`(	println  "Hello, World" )`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{"println", "Hello, World"}, matched)
		assert.NoError(t, err)
	}
	{
		remaining, _, err := p(`println "Hello, World"`)
		assert.Equal(t, `println "Hello, World"`, remaining)
		assert.EqualError(t, err, "wanted a literal \"(\", got: \"println \\\"Hello, World\\\"\"")
	}
	t.Run("recursion", func(t *testing.T) {
		t.Skip()
		remaining, matched, err := p(`(func main ())`)
		assert.Equal(t, "", remaining)
		assert.Equal(t, []interface{}{"func", "main", []interface{}{}}, matched)
		assert.NoError(t, err)
	})
}
