package main

import (
	"bytes"
	"debug/dwarf"
	"encoding/binary"
	"fmt"
	"github.com/c-bata/go-prompt"
	"go.uber.org/zap"
	"golang.org/x/arch/x86/x86asm"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// executor will exec for input.
// please keep the sync of printCmdHelper in error.go
func executor(input string) {
	logger.Debug("executor", zap.String("input", input))
	if len(input) == 0 {
		return
	}
	fs := input[0]

	cmd := target.cmd
	bp := target.bp
	bi := target.bi
	pid := int(0)
	if cmd != nil && cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	switch fs {
	case 'q':
		if input == "q" || input == "quit" {
			if cmd.Process != nil {
				if err := syscall.Kill(cmd.Process.Pid, syscall.SIGKILL); err != nil {
					// printErr(err)
				}
			}
			if os.Getenv("GODBG_TEST") != "" {
				return
			}
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
			if bInfo, err := bp.SetFileLineBreakPoint(bi, pid, filename, line); err != nil {
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
				fmt.Fprintf(stdout, "godbg add %s:%d breakpoint successfully\n", bInfo.filename, bInfo.lineno)
			}
			return
		}
		if len(sps) == 2 && (sps[0] == "bc" || sps[0] == "bclear") {
			curPc, err := getPtracePc()
			if err != nil {
				printErr(fmt.Errorf("can't find pc register, err:%s", err.Error()))
				return
			}
			if sps[1] == "all" {
				tmp := make([]*BInfo, 0, len(bp.infos))
				for _, v := range bp.infos {
					if v.kind == USERBPTYPE {
						_ = bp.disableBreakPoint(pid, v)
						if v.pc == curPc-1 {
							_ = setPcRegister(cmd, v.pc)
						}
					} else {
						tmp = append(tmp, v)
					}
				}
				bp.infos = tmp
				return
			}

			if needClearIndex, err := strconv.Atoi(sps[1]); err == nil {
				if needClearIndex > len(bp.infos) {
					printErr(fmt.Errorf("can't find breakpoint index %d", needClearIndex))
				}
				count := 0
				for i, v := range bp.infos {
					if v.kind == USERBPTYPE {
						count++
						if count == needClearIndex {
							_ = bp.disableBreakPoint(pid, v)
							if v.pc == curPc-1 {
								_ = setPcRegister(cmd, v.pc)
							}
							bp.infos = append(bp.infos[:i], bp.infos[(i+1):len(bp.infos)]...)
							_, _ = fmt.Fprintf(stdout, "clear breakpoint %d successfully, resort breakpoint again\n", needClearIndex)
							return
						}
					}
				}
				printErr(fmt.Errorf("can't find breakpoint index %d", needClearIndex))
				return
			}
		}
		if len(sps) == 1 && (sps[0] == "bl") {
			count := 0
			for _, v := range bp.infos {
				if v.kind == USERBPTYPE {
					count++
					fmt.Fprintf(stdout, "%-2d. %s:%d, pc 0x%x\n", count, v.filename, v.lineno, v.pc)
				}
			}
			if count == 0 {
				fmt.Fprintf(stdout, "there is no breakpoint\n")
			}
			return
		}
		if len(sps) == 2 && (sps[0] == "bl" && sps[1] == "all") {
			count := 0
			for _, v := range bp.infos {
				count++
				fmt.Fprintf(stdout, "%-2d. %s:%d, pc 0x%x, type %s\n", count, v.filename, v.lineno, v.pc, v.kind.String())
			}
			if count == 0 {
				fmt.Fprintf(stdout, "there is no breakpoint\n")
			}
			return
		}
		if len(sps) == 1 && sps[0] == "bt" {
			var (
				rbp      uint64
				err      error
				filename string
				line     int
				pc       uint64
				f        *Function
				ok       bool
			)
			if rbp, err = getPtraceBp(); err != nil {
				printErr(err)
				printErr(fmt.Errorf("!!1 %s", err.Error()))
				return
			}
			if pc, err = getPtracePc(); err != nil {
				printErr(err)
				printErr(fmt.Errorf("!!2 %s", err.Error()))
				return
			}

			if _, ok = bp.findBreakPoint(pc - 1); ok {
				pc = pc - 1
			}
			if filename, line, err = bi.pcTofileLine(pc); err != nil {
				printErr(err)
				printErr(fmt.Errorf("!!3 %s", err.Error()))
				return
			}
			fmt.Fprintf(stdout, "%s:%d\n", filename, line)

			ret := uint64(0)
			for {
				if uintptr(rbp) == 0 {
					break
				}
				original := make([]byte, 16)
				_, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(rbp), original)
				if err != nil {
					printErr(err)
					return
				}
				reader := bytes.NewBuffer(original)

				if err = binary.Read(reader, binary.LittleEndian, &rbp); err != nil {
					printErr(err)
					return
				}
				if err = binary.Read(reader, binary.LittleEndian, &ret); err != nil {
					printErr(err)
					return
				}
				//fmt.Fprintf(stdout, "ret = %d\n", ret)
				if filename, line, err = bi.pcTofileLine(ret - 1); err != nil {
					printErr(err)
					return
				}
				if f, err = bi.findFunctionIncludePc(ret - 1); err != nil {
					printErr(err)
					return
				}
				fmt.Fprintf(stdout, "%s:%d %s\n", filename, line, f.name)
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
			if err := bp.singleStepInstructionWithBreakpointCheck(pid); err != nil {
				printErr(err)
				return
			}
			if err := bp.Continue(pid); err != nil {
				printErr(err)
				return
			}
			var (
				s  syscall.WaitStatus
				pc uint64
			)
			wpid, err := syscall.Wait4(cmd.Process.Pid, &s, syscall.WALL, nil)
			if err != nil {
				printErr(err)
				return
			}

			if s.Exited() {
				printExit0(wpid)
				cmd.Process = nil
				return
			}

			if n := s.StopSignal(); n != syscall.SIGTRAP && n != syscall.SIGURG {
				cmd.Process = nil
				fmt.Errorf("unknown waitstatus %v, signal %d", s, s.Signal())
				return
			}

			if pc, err = getPtracePc(); err != nil {
				printErr(err)
				return
			}
			fmt.Fprintf(stdout, "current process pc = 0x%x\n", pc)
			if err = listFileLineByPtracePc(target.bi, 6); err != nil {
				printErr(err)
				return
			}
			return
		}
	case 's':
		sps := strings.Split(input, " ")
		if len(sps) == 1 && (sps[0] == "s" || sps[0] == "step") {
			var (
				err         error
				filename    string
				lineno      int
				oldfilename string
				oldlineno   int
				pc          uint64
				info        *BInfo
				ok          bool
			)
			if oldfilename, oldlineno, err = bi.getCurFileLineByPtracePc(); err != nil {
				printErr(err)
				return
			}
			for {
				if pc, err = getPtracePc(); err != nil {
					printErr(err)
					return
				}
				if filename, lineno, err = bi.pcTofileLine(pc); err != nil {
					printErr(err)
					return
				}

				if !(filename == oldfilename && lineno == oldlineno) {
					fmt.Fprintf(stdout, "current process pc = 0x%x\n", pc)
					if err = listFileLineByPtracePc(target.bi, 6); err != nil {
						printErr(err)
						return
					}
					return
				}
				if info, ok = bp.findBreakPoint(pc - 1); ok {
					if err = bp.disableBreakPoint(pid, info); err != nil {
						printErr(err)
						return
					}
					defer bp.enableBreakPoint(pid, info)
					if err = setPcRegister(target.cmd, pc-1); err != nil {
						printErr(err)
						return
					}
				}
				if err = syscall.PtraceSingleStep(cmd.Process.Pid); err != nil {
					printErr(err)
					return
				}
				var s syscall.WaitStatus
				if _, err = syscall.Wait4(cmd.Process.Pid, &s, syscall.WALL, nil); err != nil {
					printErr(err)
					return
				}
				if s.Exited() {
					printExit0(cmd.Process.Pid)
					cmd.Process = nil
					return
				}
				if n := s.StopSignal(); n != syscall.SIGTRAP && n != syscall.SIGURG {
					printErr(fmt.Errorf("unknown waitstatus %v, signal %d", s, s.Signal()))
					return
				}
			}
			return
		}
	case 'n':
		sps := strings.Split(input, " ")
		if len(sps) == 1 && (sps[0] == "n" || sps[0] == "next") {
			var (
				err         error
				pc          uint64
				info        *BInfo
				ok          bool
				filename    string
				lineno      int
				oldfilename string
				oldlineno   int

				//f *Function
				inst x86asm.Inst
			)

			if pc, err = getPtracePc(); err != nil {
				printErr(err)
				return
			}
			if info, ok = bp.findBreakPoint(pc - 1); ok {
				pc = pc - 1
				if err = setPcRegister(target.cmd, pc); err != nil {
					printErr(err)
					return
				}
				if err = bp.disableBreakPoint(pid, info); err != nil {
					printErr(err)
					return
				}
				defer bp.enableBreakPoint(pid, info)
			}
			if oldfilename, oldlineno, err = bi.pcTofileLine(pc); err != nil {
				printErr(err)
				return
			}

			calling := false
			callingfpc := uint64(0)

			for {
				if pc, err = getPtracePc(); err != nil {
					printErr(err)
					return
				}
				if info, ok = bp.findBreakPoint(pc - 1); ok {
					if err := listFileLineByPtracePc(target.bi, 6); err != nil {
						printErr(err)
						return
					}
					return
				}

				if calling == true && pc != callingfpc {
					if err = syscall.PtraceSingleStep(cmd.Process.Pid); err != nil {
						printErr(err)
						return
					}
					var s syscall.WaitStatus
					if _, err = syscall.Wait4(cmd.Process.Pid, &s, syscall.WALL, nil); err != nil {
						printErr(err)
						return
					}
					if s.Exited() {
						printExit0(cmd.Process.Pid)
						cmd.Process = nil
						return
					}
					if n := s.StopSignal(); n != syscall.SIGTRAP && n != syscall.SIGURG {
						printErr(fmt.Errorf("unknown waitstatus %v, signal %d", s, s.Signal()))
						return
					}
				} else if calling == true && pc == callingfpc {
					calling = false
					if filename, lineno, err = bi.pcTofileLine(pc); err != nil {
						printErr(err)
						return
					}
					if !(filename == oldfilename && lineno == oldlineno) {
						if err := listFileLineByPtracePc(target.bi, 6); err != nil {
							printErr(err)
							return
						}
						return
					}
				} else {
					if inst, err = bi.getSingleMemInst(cmd.Process.Pid, pc); err != nil {
						printErr(err)
						return
					}
					if inst.Op == x86asm.CALL || inst.Op == x86asm.LCALL {
						calling = true
						callingfpc = pc + uint64(inst.Len)
						continue
					}

					if err = syscall.PtraceSingleStep(cmd.Process.Pid); err != nil {
						printErr(err)
						return
					}
					var s syscall.WaitStatus
					if _, err = syscall.Wait4(cmd.Process.Pid, &s, syscall.WALL, nil); err != nil {
						printErr(err)
						return
					}
					if s.Exited() {
						printExit0(cmd.Process.Pid)
						cmd.Process = nil
						return
					}
					if n := s.StopSignal(); n != syscall.SIGTRAP && n != syscall.SIGURG {
						printErr(fmt.Errorf("unknown waitstatus %v, signal %d", s, s.Signal()))
						return
					}
					if filename, lineno, err = bi.pcTofileLine(pc + uint64(inst.Len)); err != nil {
						printErr(err)
						return
					}
					if !(filename == oldfilename && lineno == oldlineno) {
						if err := listFileLineByPtracePc(target.bi, 6); err != nil {
							printErr(err)
							return
						}
						return
					}
				}
			}
		}
	case 'l':
		sps := strings.Split(input, " ")
		if len(sps) == 1 && (sps[0] == "l" || sps[0] == "list") {
			if err := listFileLineByPtracePc(target.bi, 6); err != nil {
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
	case 'r':
		sps := strings.Split(input, " ")
		if len(sps) == 1 && (sps[0] == "r" || sps[0] == "restart") {
			pid := 0
			if cmd.Process != nil {
				pid = cmd.Process.Pid
				if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGSTOP); err != nil {
					err = nil
					//printErr(err)
					//return
				}
			}
			if pid != 0 {
				fmt.Fprintf(stdout, "  stop  old process pid %d\n", pid)
			}
			var err error
			if cmd, err = runexec(target.execFile); err != nil {
				printErr(err)
				logger.Error(err.Error(), zap.String("stage", "restart:runexec"), zap.String("execfile", target.execFile))
				return
			}
			if err = bp.SetBpWhenRestart(target.cmd.Process.Pid); err != nil {
				printErr(err)
				logger.Error(err.Error(), zap.String("stage", "restart:setbp"), zap.String("execfile", target.execFile))
				return
			}
			fmt.Fprintf(stdout, "restart new process pid %d \n", cmd.Process.Pid)
			return
		}
	case 'd':
		sps := strings.Split(input, " ")
		if len(sps) == 1 && (sps[0] == "disass" || sps[0] == "disassemble") {
			if err := listDisassembleByPtracePc(target.bi, target.bp, target.cmd.Process.Pid); err != nil {
				printErr(err)
				return
			}
			return
		}
	case 'p':
		sps := strings.Split(input, " ")
		if len(sps) == 2 && (sps[0] == "p" || sps[0] == "print") {
			var (
				v     string
				pc    uint64
				err   error
				ok    bool
				f     *Function
				frame *Frame
			)
			v = sps[1]
			if pc, err = getPtracePc(); err != nil {
				printErr(err)
				return
			}
			if _, ok = bp.findBreakPoint(pc - 1); ok {
				pc--
			}
			if frame, err = bi.findFrameInformation(pc); err != nil {
				printErr(err)
				return
			}
			if f, err = bi.findFunctionIncludePc(pc); err != nil {
				printErr(err)
				return
			}
			for _, fv := range f.variables {
				isFound := false
				if field := fv.AttrField(dwarf.AttrName); field != nil {
					if fieldstr, ok := field.Val.(string); ok && fieldstr == v {
						isFound = true
					}
				}
				if isFound {
					var (
						opcode byte
					)
					field := fv.AttrField(dwarf.AttrLocation)
					buf := bytes.NewBuffer(field.Val.([]byte))
					if opcode, err = buf.ReadByte(); err != nil {
						printErr(err)
						return
					}
					switch opcode {
					case DW_OP_fbreg:
						num, _, _ := DecodeSLEB128(buf)
						address := int64(frame.framebase) + num
						// if the type is `string`
						val := make([]byte, 8)
						if _, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(address)+uintptr(8), val); err != nil {
							printErr(err)
							return
						}
						strlen := int64(binary.LittleEndian.Uint64(val))
						if strlen < 0 {
							printErr(fmt.Errorf("strlen %d shoulde be < 0", strlen))
							return
						}
						// read addr
						if _, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(address), val); err != nil {
							printErr(err)
							return
						}
						addr := uintptr(binary.LittleEndian.Uint64(val))
						if addr == 0 {
							printErr(fmt.Errorf("pointer addr %d shoulde be == 0", addr))
							return
						}
						logger.Debug(fmt.Sprintf("address = %d,  len = %d, addr = %d,, num = %d\n", address, strlen, addr, num))

						strpointer := make([]byte, strlen)
						if _, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(addr), strpointer); err != nil {
							printErr(err)
							return
						}
						fmt.Fprintf(stdout, "%v\n", *(*string)(unsafe.Pointer(&strpointer)))
						return
					}
					fmt.Fprintf(stderr, "not support dwarf variable %#v", fv)
				}
			}
			return
		}
	case 'h':
		sps := strings.Split(input, " ")
		if len(sps) == 1 && (sps[0] == "h" || sps[0] == "help") {
			printCmdHelper()
			return
		}
	}
	printUnsupportCmd(input)
}

func complete(docs prompt.Document) []prompt.Suggest {
	sps := strings.Split(docs.Text, " ")

	s := make([]prompt.Suggest, 0)

	curWd, _ := os.Getwd()

	if len(sps) == 2 {
		if sps[0] == "b" || sps[0] == "break" || sps[0] == "l" || sps[0] == "list" {
			for filename := range target.bi.Sources {
				if strings.HasPrefix(filename, sps[1]) {
					if filename[0] == '/' {
						filename = filename[1:]
					}
					s = append(s, prompt.Suggest{Text: filename, Description: ""})
				} else {

					inputPrefix := sps[1]
					if inputPrefixFilename := path.Join(curWd, inputPrefix); strings.HasPrefix(filename, inputPrefixFilename) {
						needComplete := ""
						if inputPrefix == "./" {
							inputPrefix = ""
							needComplete = filename[len(inputPrefixFilename):]
							if len(needComplete) > 0 && needComplete[0] == '/' {
								needComplete = needComplete[1:]
							}
						} else {
							if len(inputPrefix) > 2 && inputPrefix[:2] == "./" {
								inputPrefix = inputPrefix[2:]
							}
							needComplete = filename[len(inputPrefixFilename):]
						}
						s = append(s, prompt.Suggest{Text: inputPrefix + needComplete, Description: ""})
					}
				}
				if len(s) >= 30 {
					return s
				}
			}
		}
	}
	return s
}

const (
	_AT_NULL_AMD64  = 0
	_AT_ENTRY_AMD64 = 9
)

func entryPointFromAuxvAMD64(auxv []byte) uint64 {
	rd := bytes.NewBuffer(auxv)

	for {
		var tag, val uint64
		err := binary.Read(rd, binary.LittleEndian, &tag)
		if err != nil {
			return 0
		}
		err = binary.Read(rd, binary.LittleEndian, &val)
		if err != nil {
			return 0
		}

		switch tag {
		case _AT_NULL_AMD64:
			return 0
		case _AT_ENTRY_AMD64:
			return val
		}
	}
}
