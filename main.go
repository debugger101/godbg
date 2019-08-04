package main

import (
	"github.com/c-bata/go-prompt"
	"github.com/chainhelen/godbg/log"
	"go.uber.org/zap"
	"os"
	"os/exec"
)

var logger = log.Log
var bp = &BP{}
var (
	bi *BI
	cmd *exec.Cmd
)

func main() {
	var (
		filename string
		execfile string
		err      error
		p *prompt.Prompt
	)

	if err = checkArgs(); err != nil {
		logger.Error(err.Error(), zap.String("stage","checkArgs"), zap.Strings("args", os.Args))
		printHelper()
		return
	}

	// step 1, get absolute filename
	if filename, err = absoluteFilename(); err != nil {
		logger.Error(err.Error(), zap.String("stage","absolute"), zap.String("filename", filename))
		printHelper()
		return
	}

	// step 2, build the filename into executable file
	if execfile, err = build(filename); err != nil {
		logger.Error(err.Error(), zap.String("stage", "build"),zap.String("filename", filename))
		printHelper()
		return
	}
	defer os.Remove(execfile)

	// step 3, analyze executable file; The most import places are "_debug_info", "_debug_line"
	if bi, err = analyze(execfile);err != nil {
		logger.Error(err.Error(), zap.String("stage", "analyze"),
			zap.String("filename", filename), zap.String("execfile", execfile))
		printHelper()
		return
	}

	// step 4, run executable file
	if cmd, err = runexec(execfile); err != nil {
		logger.Error(err.Error(), zap.String("stage", "runexec"),
			zap.String("filename", filename), zap.String("execfile", execfile))
		printHelper()
		return
	}

	// step 5, run prompt. `executor` handle all input
	p = prompt.New(
		executor,
		complete,
		prompt.OptionTitle("Simplified golang debugger"),
		prompt.OptionPrefix("(godbg) "),
		prompt.OptionInputTextColor(prompt.Yellow),
	)
	p.Run()
}
