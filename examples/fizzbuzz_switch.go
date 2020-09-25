package main

import "fmt"

func main() {
	for i := 0; i < 100; i++ {
		switch {
		case 0 == i%15:
			fmt.Println("fizzbuzz")
		case 0 == i%3:
			fmt.Println("fizz")
		case 0 == i%5:
			fmt.Println("buzz")
		default:
			fmt.Println(i)
		}
	}
}
