package main

import "time"

func main() {
	println(time.Now().Add(time.Second).Unix())
}
