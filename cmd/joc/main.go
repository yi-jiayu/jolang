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
	output, err := jo.Compile(string(all))
	if err != nil {
		panic(err)
	}
	fmt.Println(output)
}
