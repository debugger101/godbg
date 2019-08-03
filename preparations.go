package main

import (
	"errors"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"syscall"
)

func checkArgs() error {
	logger.Debug("[checkArgs]", zap.Strings("args", os.Args))
	if len(os.Args) != 3 {
		return errors.New("len(args) != 3")
	}
	debug := os.Args[1]

	if debug != "debug" {
		return errors.New("only support `debug`")
	}
	if  path.Ext(os.Args[2]) != ".go" {
		return errors.New("please input .go file")
	}
	return nil
}

func absoluteFilename() (string, error) {
	filename := os.Args[2]
	if path.IsAbs(filename) {
		return filename, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	filename = path.Join(dir, filename)
	_, err = os.Stat(filename)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func build (filename string) (string, error) {
	base := filepath.Base(filename)
	execfile := path.Join(os.TempDir(), "__" + base+"__")

	args := []string{"build", "-gcflags", "all=-N -l", "-o", execfile, filename}

	cmd := exec.Command("go", args...)
	return execfile, cmd.Run()
}

// not supoort arguments of cmds
func runexec(execfile string) (*exec.Cmd, error){
	cmd := exec.Command(execfile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Ptrace: true, Setpgid: true, Foreground: false}

	// !!! maybe the diffrences of routine and thread in golang
	runtime.LockOSThread()

	if err := cmd.Start(); err != nil {
		logger.Error("runexec:cmd.Start()", zap.Error(err))
		return nil, err
	}

	if err := cmd.Wait(); err != nil && err.Error() != "stop signal: trace/breakpoint trap" {
		logger.Error("runexec:cmd.Wait()", zap.Error(err))
		return nil, err
	}
	return cmd, nil
}

