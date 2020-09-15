package jo

import (
	"go/ast"
)

func Parse(input string) (*ast.File, error) {
	_, node, err := SourceFile(input)
	if err != nil {
		return nil, err
	}
	return node.(*ast.File), nil
}
