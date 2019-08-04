package main

import "syscall"

func getRegisters() (syscall.PtraceRegs, error){
	var prs syscall.PtraceRegs
	if cmd.Process == nil {
		return prs, NoProcessRuning
	}
	err := syscall.PtraceGetRegs(cmd.Process.Pid, &prs)
	return prs, err
}

func getPtracePc() (uint64, error) {
	var (
		prs syscall.PtraceRegs
		err error
	)
	if prs, err = getRegisters(); err != nil {
		return 0, err
	}
	return prs.PC(), nil
}