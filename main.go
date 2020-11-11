package main

import (
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
	"github.com/debugger101/godbg/log"
	"go.uber.org/zap"
	"io"
	"os"
)

var (
	target *Target
	logger = log.Log

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
)

func main() {
	var (
		filename string
		err      error
		p        *prompt.Prompt
	)
	stdin = os.Stdin
	stdout = os.Stdout
	stderr = os.Stderr
	target = &Target{bi: &BI{}, bp: &BP{}}

	if err = checkArgs(); err != nil {
		logger.Error(err.Error(), zap.String("stage", "checkArgs"), zap.Strings("args", os.Args))
		printExecutableProgramHelper()
		return
	}

	// step 1, get absolute filename
	if filename, err = absoluteFilename(); err != nil {
		logger.Error(err.Error(), zap.String("stage", "absolute"), zap.String("filename", filename))
		printExecutableProgramHelper()
		return
	}

	// step 2, build the filename into executable file
	if target.execFile, err = build(filename); err != nil {
		logger.Error(err.Error(), zap.String("stage", "build"), zap.String("filename", filename))
		printExecutableProgramHelper()
		return
	}
	defer os.Remove(target.execFile)

	// step 3, analyze executable file; The most import places are "_debug_info", "_debug_line"
	if target.bi, err = analyze(target.execFile); err != nil {
		logger.Error(err.Error(), zap.String("stage", "analyze"),
			zap.String("filename", filename), zap.String("execfile", target.execFile))
		printExecutableProgramHelper()
		return
	}

	// step 4, run executable file
	if target.cmd, err = runexec(target.execFile); err != nil {
		logger.Error(err.Error(), zap.String("stage", "runexec"),
			zap.String("filename", filename), zap.String("execfile", target.execFile))
		printExecutableProgramHelper()
		return
	}
	fmt.Fprintf(stdout, "trace cur process pid %d\n", target.cmd.Process.Pid)

	// step 5, run prompt. `executor` handle all input
	p = prompt.New(
		executor,
		complete,
		prompt.OptionTitle("Simplified golang debugger"),
		prompt.OptionPrefix("(godbg) "),
		prompt.OptionInputTextColor(prompt.Yellow),
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator),
	)
	p.Run()
}
