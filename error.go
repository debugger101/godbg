package main

import (
	"errors"
	"fmt"
)

var NotFoundSourceLineErr = errors.New("cant't find this source line")
var HasExistedBreakPointErr = errors.New("this breakpoint has existed")
var NoProcessRuning = errors.New("there is no process running")

type NotFoundFuncErr struct {
	pc uint64
}

func (e *NotFoundFuncErr) Error() string {
	return fmt.Sprintf("findFunctionIncludePc can't find function by pc:%d", e.pc)
}

func printHelper() {
	fmt.Fprintf(stderr, "%s\n", "Usage:\n\tJust like ./godgb debug main.go")
}

func printUnsupportCmd(cmd string) {
	fmt.Fprintf(stderr,"unsupport cmd `%s`\n", cmd)
}

func printHasExistedBreakPoint(place string) {
	fmt.Fprintf(stderr,"existed breakpoint %s\n", place)
}

func printNotFoundSourceLineErr(place string) {
	fmt.Fprintf(stderr, "can't find this source line %s\n", place)
}

func printErr(err error) {
	fmt.Fprintf(stderr,"%s\n", err.Error())
}

func printNoProcessErr() {
	fmt.Fprintf(stderr,"%s\n", NoProcessRuning.Error())
}

func printExit0(opid int) {
	fmt.Fprintf(stderr,"Process %d has exited with status 0\n", opid)

}