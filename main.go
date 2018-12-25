package main

import (
	"fmt"
	"github.com/peterh/liner"
	"log"
	"os"
	"os/exec"
	"strings"
)

var logger *log.Logger

func prihelper() {
	fmt.Printf("%s\n", "Just like ./dgb debug main.go")
}

func build(debugname string, targetname string) error {
	args := []string{"build", "-gcflags", "all=-N -l", "-o", debugname, targetname}
	logger.Printf("[build] go %s\n", strings.Join(args, " "))
	goBuild := exec.Command("go", args...)
	goBuild.Stderr = os.Stderr
	return goBuild.Run()
}

func remove(path string) {
	logger.Printf("[remove] path:%s\n", path)
	err := os.Remove(path)
	if err != nil {
		logger.Printf("[remove]could not remove %v: %v\n", path, err)
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
	if err := build(os.Args[1], os.Args[2]); err != nil {
		panic(err)
	}
	return os.Args[1]
}

func run(debugFile string) {
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
		logger.Printf("[run] cmdstr:%v\n", cmdstr)
		switch cmdstr {
		case "q":
			logger.Printf("[run] receive \"q\", quit\n")
			fmt.Printf("bye\n")
			quit = true
		default:
			fmt.Printf("Wrong input, please document\n")
			logger.Printf("[run] receive default\n")
		}

		if quit {
			break
		}
	}
}

func initLog() {
	logPath := os.Getenv("DBGLOG")
	log.Printf("[initLog] os.Getenv(\"DBGLOG\"):%s\n", logPath)
	if "stdout" == strings.TrimSpace(logPath) {
		logger = log.New(os.Stdout, "[gomindbg]", log.LstdFlags|log.Lshortfile)
		return
	}
	if "" == strings.TrimSpace(logPath) {
		logPath = "/dev/null"
	}
	f, e := os.OpenFile(logPath, os.O_RDWR, 0)
	if e != nil {
		panic(e)
	}
	// defer f.Close()
	logger = log.New(f, "[gomindbg]", log.LstdFlags|log.Lshortfile)
}

func main() {
	initLog()
	debugFile := checkArgsAndBuild()
	defer remove(debugFile)

	run(debugFile)
}
