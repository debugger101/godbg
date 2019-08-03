package main

import (
	"fmt"
	"github.com/c-bata/go-prompt"
	"go.uber.org/zap"
	"os"
	"strings"
	"syscall"
)

func executor(input string) {
	logger.Debug("executor", zap.String("input", input))
	if len(input) == 0 {
		return
	}
	fs := input[0]

	switch fs {
	case 'q':
		if input == "q" || input == "quit"{
			os.Exit(0)
		}
	case 'b':
		sps := strings.Split(input, " ")
		if len(sps) == 2 && (sps[0] == "b" || sps[0] == "break") {
			filename, line, err := parseLoc(sps[1])
			if err != nil {
				printUnsupportCmd(input)
				return
			}
			if bInfo, err := bp.SetFileLineBreakPoint(filename, line); err != nil {
				if err == HasExistedBreakPointErr {
					printHasExistedBreakPoint(sps[1])
					return
				}
				if err == NotFoundSourceLineErr {
					printNotFoundSourceLineErr(sps[1])
					return
				}
				printErr(err)
				return
			} else {
				fmt.Printf("godbg add %s:%d breakpoint successfully\n",bInfo.filename, bInfo.lineno)
			}
			return
		}
	case 'c':
		sps := strings.Split(input, " ")
		if len(sps) == 1 && (sps[0] == "c" || sps[0] == "continue") {

			if cmd.Process == nil {
				printNoProcessErr()
				return
			}

			if err := bp.Continue(); err != nil {
				printErr(err)
				return
			}

			var s syscall.WaitStatus
			wpid, err := syscall.Wait4(cmd.Process.Pid, &s, syscall.WALL, nil)
			if err != nil {
				printErr(err)
				return
			}
			status := (syscall.WaitStatus)(s)
			if status.Exited() {
				// TODO
				if cmd.Process != nil && wpid == cmd.Process.Pid {
					printExit0(wpid)
				} else {
					printExit0(wpid)
				}
				cmd.Process = nil
			}
			return
		}
	}
	printUnsupportCmd(input)
}

func complete(docs prompt.Document) []prompt.Suggest {
	_ = docs
	return nil
}
