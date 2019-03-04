package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/peterh/liner"
	"godbg/bininfo"
	"godbg/bp"
	"godbg/log"
	alog "log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const (
	_AT_NULL_AMD64  = 0
	_AT_ENTRY_AMD64 = 9
)

var logger *alog.Logger

func init() {
	logger = log.Logger
}

func prihelper() {
	fmt.Printf("%s\n", "Just like ./dgb debug main.go")
}

func build(debugname string, targetname string) error {
	args := []string{"build", "-gcflags", "all=-N -l", "-o", debugname, targetname}
	logger.Printf("\t[build] go %s\n", strings.Join(args, " "))
	goBuild := exec.Command("go", args...)
	goBuild.Stderr = os.Stderr
	return goBuild.Run()
}

func remove(path string) {
	logger.Printf("\t[remove] path:%s\n", path)
	err := os.Remove(path)
	if err != nil {
		logger.Printf("\t[remove]could not remove %v: %v\n", path, err)
	}
}

func checkArgsAndBuild() string {
	arg_num := len(os.Args)
	if arg_num != 3 {
		prihelper()
		panic(fmt.Sprintf("Wrong args: Expect the length of args is 3, but get %d", arg_num))
	}
	//if os.Args[0] != "gomindbg" {
	//	prihelper()
	//	panic(fmt.Sprintf("Wrong args: Expect the first argument is not \"gomindbg\", but get %s", os.Args[0]))
	//}
	if os.Args[1] != "debug" {
		prihelper()
		panic(fmt.Sprintf("Wrong args: Expect the second argument is not \"debug\", but get %s", os.Args[1]))
	}
	var (
		debugName string
		err       error
	)
	debugName, err = filepath.Abs(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err = build(debugName, os.Args[2]); err != nil {
		panic(err)
	}
	return debugName
}

func CliPutDoc() {
	fmt.Printf("Wrong input, please read document\n")
	logger.Printf("\t[run] receive default\n")
}

func cliRun(process *os.Process, bi *bininfo.BinaryInfo) {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	bpc := &bp.BP{}

	for {
		var quit bool
		var err error
		var cmdstr string
		if cmdstr, err = line.Prompt("(dbg)"); err != nil {
			panic(err)
		}
		cmdstr = strings.TrimSpace(cmdstr)
		logger.Printf("\t[run] cmdstr:%v\n", cmdstr)

		if len(cmdstr) <= 0 {
			CliPutDoc()
		} else {
			switch cmdstr[0] {
			case 'q':
				if len(cmdstr) != 1 && (cmdstr) != "quit" {
					CliPutDoc()
					break
				}
				logger.Printf("\t[run] receive \"q\", quit\n")
				logger.Printf("\t[run] bye")
				quit = true
			case 'f':
				logger.Printf("\t[run] receive \"f\", find\n")
				splitstrs := strings.Split(cmdstr, " ")
				if len(splitstrs) != 2 {
					CliPutDoc()
					break
				}
				findstr := splitstrs[0]
				locstr := splitstrs[1]
				if len(findstr) != 1 && (findstr) != "find" {
					CliPutDoc()
					break
				}
				filelocstr := strings.Split(locstr, ":")
				if len(filelocstr) != 2 {
					CliPutDoc()
					break
				}
				if add, err := bi.FindLocationFromFileLoc(filelocstr[0], filelocstr[1]); err != nil {
					fmt.Printf("Find Result something error: %s\n", err.Error())
					break
				} else {
					fmt.Printf("Find Result as Addr: %#v\n", add)
					break
				}
			case 'b':
				logger.Printf("\t[run] receive \"b\", breakpoint\n")
				splitstrs := strings.Split(cmdstr, " ")
				if len(splitstrs) != 2 {
					CliPutDoc()
					break
				}
				bpstr := splitstrs[0]
				locstr := splitstrs[1]
				if len(bpstr) != 1 && (bpstr) != "breakpoint" {
					CliPutDoc()
					break
				}
				filelocstr := strings.Split(locstr, ":")
				if len(filelocstr) != 2 {
					CliPutDoc()
					break
				}
				var (
					add uint64
					err error
				)
				if add, err = bi.FindLocationFromFileLoc(filelocstr[0], filelocstr[1]); err != nil {
					fmt.Printf("Find Result something error: %s\n", err.Error())
					break
				} else {
					fmt.Printf("Find Result as Addr: %#v\n", add)
				}
				bpInfo := &bp.Info{
					Pid:        process.Pid,
					Original:   make([]byte, 1),
					Filename:   filelocstr[0],
					Filelineno: filelocstr[1],
					Addr:       uintptr(add),
				}

				if err = bpc.SetBreakpoint(bpInfo); err != nil {
					fmt.Printf("Set pid:%d addr:%d err:%s\n", process.Pid, add, err.Error())
					break
				}
				fmt.Printf("Set %s breakpoaint sucesullfy\n", locstr)
			case 'c':
				if err = bp.Continue(process.Pid); err != nil {
					fmt.Printf("Continue Pid:%d failed\n", process.Pid)
					break
				}
				fmt.Printf("Continue Pid:%d sucessfully\n", process.Pid)
			case 'l':
				cmdTrimStr := strings.TrimSpace(cmdstr)
				if cmdTrimStr == "l b" || cmdTrimStr == "l breakpoints" {
					bpc.ListBreakpoint()
				} else if cmdTrimStr == "l s" || cmdTrimStr == "l sources" {
					var (
						pc uint64
					)
					if pc, err = bpc.GetPC(process.Pid); err != nil {
						fmt.Printf("l s failed, err:%s\n", err.Error())
						break
					}
					_ = pc
				} else if cmdTrimStr == "l r" || cmdTrimStr == "l regs" {
					if err = bpc.ListRegs(process.Pid); err != nil {
						fmt.Printf("[error] list regs err:%s\n", err.Error())
					}
				} else {
					CliPutDoc()
					break
				}
			default:
				CliPutDoc()
				break
			}
		}
		if quit {
			break
		}
	}
}

func launch(cmd []string) *os.Process {
	var (
		execmd *exec.Cmd
		err    error
	)

	// copy from dlv:  check that the argument to Launch is an executable file
	if fi, staterr := os.Stat(cmd[0]); staterr == nil && (fi.Mode()&0111) == 0 {
		logger.Printf("\t[launch] can't " + err.Error())
		panic(err)
	}

	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! should be existed
	runtime.LockOSThread()

	execmd = exec.Command(cmd[0])
	execmd.Stdout = os.Stdout
	execmd.Stderr = os.Stderr
	//execmd.SysProcAttr = &syscall.SysProcAttr{Ptrace: true, Setpgid: true, Foreground: false}
	execmd.SysProcAttr = &syscall.SysProcAttr{Ptrace: true, Setpgid: true, Foreground: false}

	err = execmd.Start()
	if err != nil {
		logger.Printf("\t[launch] %s", err.Error())
		panic(err)
	}
	logger.Printf("\t[launch] Waiting for exec:%s, pid:%d to finish\n", strings.Join(cmd, ""), execmd.Process.Pid)

	// shouldn't panic
	//if err != nil {
	//	logger.Printf("\t[launch] %s\n", err.Error())
	//	panic(err)
	//}
	err = execmd.Wait()
	if err != nil {
		logger.Printf("\t[launch] childre execmd get %s\n", err.Error())
	}
	return execmd.Process
}

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

func main() {
	debugFile := checkArgsAndBuild()
	defer remove(debugFile)

	process := launch([]string{debugFile})
	bi := bininfo.LoadBinInfo(debugFile, process)

	cliRun(process, bi)
}
