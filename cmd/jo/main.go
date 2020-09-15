package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/yi-jiayu/jo"
)

func main() {
	all, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	_, matched, err := jo.SExprs()(string(all))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", matched)
}
