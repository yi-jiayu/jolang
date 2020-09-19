package main

import (
	"fmt"
	"strconv"
)

func main() {
	s := "hello"
	_, err := strconv.Atoi(s)
	if err != nil {
		fmt.Printf("error: not a number: %s\n", s)
	}
}
