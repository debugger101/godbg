package main

import "fmt"

var tmps string

func ptest() {
	tmps = "tmp s"
	fmt.Printf("hello test\n")
	fmt.Printf("%s\n", tmps)
}

func main() {
	fmt.Printf("hello world\n")
	ptest()
}
