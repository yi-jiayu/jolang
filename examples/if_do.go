package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	if rand.Float32() < 0.5 {
		fmt.Println("heads")
		fmt.Println("not tails")
	}
}
