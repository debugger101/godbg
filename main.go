package main

import (
	"fmt"
	"github.com/peterh/liner"
	"godbg/log"
	alog "log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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

func cliRun() {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	for {
		var quit bool
		var err error
		var cmdstr string
		if cmdstr, err = line.Prompt("(dbg)"); err != nil {
			panic(err)
		}
		cmdstr = strings.TrimSpace(cmdstr)
		logger.Printf("\t[run] cmdstr:%v\n", cmdstr)
		switch cmdstr {
		case "q":
			logger.Printf("\t[run] receive \"q\", quit\n")
			logger.Printf("\t[run] bye")
			quit = true
		default:
			fmt.Printf("Wrong input, please document\n")
			logger.Printf("\t[run] receive default\n")
		}

		if quit {
			break
		}
	}
}

func launch(cmd []string) {
	var (
		process *exec.Cmd
		err     error
	)

	// copy from dlv:  check that the argument to Launch is an executable file
	if fi, staterr := os.Stat(cmd[0]); staterr == nil && (fi.Mode()&0111) == 0 {
		logger.Printf("\t[launch] can't " + err.Error())
		panic(err)
	}

	// don't konw LockOSThread()
	// runtime.LockOSThread()

	process = exec.Command(cmd[0])
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	//process.SysProcAttr = &syscall.SysProcAttr{Ptrace: true, Setpgid: true, Foreground: false}
	process.SysProcAttr = &syscall.SysProcAttr{Ptrace: true, Foreground: false}

	err = process.Start()
	if err != nil {
		logger.Printf("\t[launch] %s", err.Error())
		panic(err)
	}
	logger.Printf("\t[launch] Waiting for %s to finish\n", strings.Join(cmd, ""))

	// shouldn't panic
	//if err != nil {
	//	logger.Printf("\t[launch] %s\n", err.Error())
	//	panic(err)
	//}
	err = process.Wait()
	if err != nil {
		logger.Printf("\t[launch] childre process get %s\n", err.Error())
	}
}

func main() {
	debugFile := checkArgsAndBuild()
	defer remove(debugFile)

	launch([]string{debugFile})
	cliRun()
}
