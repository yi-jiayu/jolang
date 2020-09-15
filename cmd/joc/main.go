package main

import (
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"

	"github.com/yi-jiayu/jo"
)

func main() {
	all, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	ast, err := jo.Parse(string(all))
	if err != nil {
		panic(err)
	}
	printer.Fprint(os.Stdout, token.NewFileSet(), ast)
}
