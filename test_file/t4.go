package main

import "fmt"

func pppp2(m int) string {
	return fmt.Sprintf("m = %d", m)
}

func pppp1(n, m int) {
	fmt.Printf("n = %d\n", n)
	mstr := pppp2(m)
	fmt.Println(mstr)
}

func main() {
	pppp1(200, 300)
}
