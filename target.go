package main

import (
	"os/exec"
)

type Target struct {
	bp       *BP
	bi       *BI
	cmd      *exec.Cmd
	execFile string
}
