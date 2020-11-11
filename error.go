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

func printExecutableProgramHelper() {
	fmt.Fprintf(stderr, "%s\n", "Usage:\n\tJust like `godbg debug ./main.go`.\n\tThe `main.go` is the file which you want debug.")
}

// printCmdHelper print all usages of cmd.
// please keep the sync of prompt.go.
// TODO, 1. to generate the md for user;  2. i8n.
func printCmdHelper() {
	fmt.Fprintf(stderr, "Usage:\n"+
		"\t q  (quit)                   ----   quit the debugger.\n"+
		"\t b  (break) <filename:line>  ----   set an breakpoint at specific the line of filename.\n"+
		"\t bc (bclear) all             ----   clear all breakpoints.\n"+
		"\t bl [all]                    ----   list all breakpoints if `all`.\n"+
		"\t bt                          ----   show call stack.\n"+
		"\t c  (continue)               ----   continue the paused programe.\n"+
		"\t s  (step)                   ----   step one instruction.\n"+
		"\t n  (next)                   ----   next step for source code.\n"+
		"\t l  (list) <filename:line>   ----   show the code for specific the line of filename.\n"+
		"\t r  (restart)                ----   restart the traced programe.\n"+
		"\t disass (disassemble)        ----   show the asm at cur breakpoint.\n"+
		"\t p  (print) <varibale>       ----   print the variable.but just support string type for now.\n"+
		"\t h  (help)                   ----   show the usage for cmd.\n")
}

func printUnsupportCmd(cmd string) {
	fmt.Fprintf(stderr, "unsupport cmd `%s`\n", cmd)
}

func printHasExistedBreakPoint(place string) {
	fmt.Fprintf(stderr, "existed breakpoint %s\n", place)
}

func printNotFoundSourceLineErr(place string) {
	fmt.Fprintf(stderr, "can't find this source line %s\n", place)
}

func printErr(err error) {
	fmt.Fprintf(stderr, "%s\n", err.Error())
}

func printNoProcessErr() {
	fmt.Fprintf(stderr, "%s\n", NoProcessRuning.Error())
}

func printExit0(opid int) {
	fmt.Fprintf(stderr, "Process %d has exited with status 0\n", opid)

}
