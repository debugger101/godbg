package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/c-bata/go-prompt"
	"go.uber.org/zap"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func listFileLineByPtracePc(rangeline int) error {
	pc, err := getPtracePc()
	if err != nil{
		return err
	}
	filename, lineno, err := bi.pcTofileLine(pc)
	if err != nil {
		return err
	}
	logger.Debug("list", zap.Int(filename, lineno))
	return listFileLine(filename, lineno, rangeline)
}

func listFileLine(filename string, lineno int, rangeline int) error{
	rangeMin := lineno - rangeline - 1
	rangeMax := lineno + rangeline - 1

	if rangeMin < 1 {
		rangeMin = 1
	}

	if rangeMax - rangeMin <= 0 {
		return errors.New("not right linenoe or rangeline")
	}

	file, err := os.OpenFile(filename, os.O_RDONLY, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	listFileLineBytesSlice := make([]string, 0, rangeMax - rangeMin + 2)

	listFileLineBytesSlice = append(listFileLineBytesSlice, fmt.Sprintf("list %s:%d\n", filename, lineno))
	var curLine int
	for {
		curLine++
		if curLine > rangeMax {
			break
		}

		lineBytes, err := reader.ReadSlice('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if rangeMin <= curLine && curLine <= rangeMax {
			if curLine == lineno {
				lineBytes = append([]byte(fmt.Sprintf("==>%7d: ", curLine)), lineBytes...)
			} else {
				lineBytes = append([]byte(fmt.Sprintf("   %7d: ", curLine)), lineBytes...)
			}
			listFileLineBytesSlice = append(listFileLineBytesSlice, string(lineBytes))
		}
	}

	fmt.Println(strings.Join(listFileLineBytesSlice, ""))

	return nil
}

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
				return
			}
			if err = listFileLineByPtracePc(6); err != nil {
				printErr(err)
				return
			}
			return
		}
	case 'l':
		sps := strings.Split(input, " ")
		if len(sps) == 1 && (sps[0] == "l" || sps[0] == "list") {
			if err := listFileLineByPtracePc(6); err != nil {
				printErr(err)
				return
			}
			return
		}

		if len(sps) == 2 && (sps[0] == "l" || sps[0] == "list") {
			filename, line, err := parseLoc(sps[1])
			if err != nil {
				printUnsupportCmd(input)
				return
			}
			if err = listFileLine(filename, line, 6); err != nil {
				printErr(err)
				return
			}
			return
		}

		if len(sps) == 3 && (sps[0] == "l" || sps[0] == "list") {
			filename, line, err := parseLoc(sps[1])
			if err != nil {
				printUnsupportCmd(input)
				return
			}
			rangeline, err := strconv.Atoi(sps[2])
			if err != nil {
				printUnsupportCmd(input)
				return
			}
			if err = listFileLine(filename, line, rangeline); err != nil {
				printErr(err)
				return
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
