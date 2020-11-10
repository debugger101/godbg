package main

import "fmt"

type A struct {
	a string
}

func main() {
	godbgvstr := "hello world"
	godbgvint := uint64(100)
	godbgvstruct := A{a: "hello a"}

	fmt.Printf("%s %d %v\n", godbgvstr, godbgvint, godbgvstruct)
}
