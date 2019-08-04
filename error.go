package main

import (
	"errors"
	"fmt"
)

var NotFoundSourceLineErr = errors.New("cant't find this source line")
var HasExistedBreakPointErr = errors.New("this breakpoint has existed")
var NoProcessRuning = errors.New("there is no process running")

func printHelper() {
	fmt.Printf("%s\n", "Usage:\n\tJust like ./godgb debug main.go")
}

func printUnsupportCmd(cmd string) {
	fmt.Printf("unsupport cmd `%s`\n", cmd)
}

func printHasExistedBreakPoint(place string) {
	fmt.Printf("existed breakpoint %s\n", place)
}

func printNotFoundSourceLineErr(place string) {
	fmt.Printf("can't find this source line %s\n", place)
}

func printErr(err error) {
	fmt.Printf("%s\n", err.Error())
}

func printNoProcessErr() {
	fmt.Printf("%s\n", NoProcessRuning.Error())
}

func printExit0(opid int) {
	fmt.Printf("Process %d has exited with status 0\n", opid)

}