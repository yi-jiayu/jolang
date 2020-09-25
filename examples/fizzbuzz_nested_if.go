package main

import "fmt"

func main() {
	for i := 0; i < 100; i++ {
		if 0 == i%15 {
			fmt.Println("fizzbuzz")
		} else {
			if 0 == i%3 {
				fmt.Println("fizz")
			} else {
				if 0 == i%5 {
					fmt.Println("buzz")
				} else {
					fmt.Println(i)
				}
			}
		}
	}
}
