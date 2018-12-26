package log

import (
	"log"
	"os"
	"strings"
)

var Logger *log.Logger

func init() {
	logPath := os.Getenv("DBGLOG")
	log.Printf("[initLog] os.Getenv(\"DBGLOG\"):%s\n", logPath)
	if "stdout" == strings.TrimSpace(logPath) {
		Logger = log.New(os.Stdout, "[gomindbg]", log.LstdFlags|log.Lshortfile)
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
	Logger = log.New(f, "[gomindbg]", log.LstdFlags|log.Lshortfile)
}
