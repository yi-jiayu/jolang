package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	r := bufio.NewReader(os.Stdin)
	fmt.Print("Your guess: ")
	text, _ := r.ReadString('\n')
	var guess int
	fmt.Sscan(text, &guess)
	fmt.Printf("You guessed %d!\n", guess)
}
