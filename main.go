package main

import (
	"fmt"
	"github.com/chainhelen/godbg/log"
	"go.uber.org/zap"
	"os"
	"path"
)

func printHelper() {
	fmt.Printf("%s\n", "Just like ./godgb debug main.go")
}

func checkArgs() bool {
	if len(os.Args) != 3 {
		return false
	}
	debug := os.Args[1]

	if debug != "debug" {
		return false
	}
	if path.Ext("") != "go" {
		return false
	}
	return true
}

func absoulteFilename () (string, error) {
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
	return filename,nil
}

var logger = log.Log

func main() {
	var (
		filename string
		err error
	)

	if false == checkArgs() {
		printHelper()
		return
	}

	if 	filename, err = absoulteFilename(); err != nil {
		logger.Error(err.Error(), zap.String("filename", filename))
		printHelper()
		return
	}
}
