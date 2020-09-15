package jo

import (
	"go/ast"
)

var parse = SourceFile()

func Parse(input string) (*ast.File, error) {
	_, node, err := parse(input)
	if err != nil {
		return nil, err
	}
	return node.(*ast.File), nil
}
