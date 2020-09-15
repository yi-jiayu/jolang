# jolang
Transpiling S-expressions to Go code

## About

Jolang is an experiment with producing a Lisp dialect which compiles directly to Go source code.

## Examples

minimal.jo:
```
(package main)

(func main () (println "Hello, World"))
```

minimal.go:
```
package main

func main() {
	println("Hello, World")
}
```

integers.jo:
```
(package main)

(func main () (println "this is an integer" 1))
```

integers.go:
```
package main

func main() {
	println("this is an integer", 1)
}
```
