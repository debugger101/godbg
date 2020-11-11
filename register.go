package main

import (
	"os/exec"
	"syscall"
)

func getRegisters(cmd *exec.Cmd) (syscall.PtraceRegs, error) {
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
	if prs, err = getRegisters(target.cmd); err != nil {
		return 0, err
	}
	return prs.PC(), nil
}

func setPcRegister(cmd *exec.Cmd, pc uint64) error {
	var (
		prs syscall.PtraceRegs
		err error
	)
	if prs, err = getRegisters(target.cmd); err != nil {
		return err
	}
	prs.SetPC(pc)
	return syscall.PtraceSetRegs(cmd.Process.Pid, &prs)
}

func getPtraceBp() (uint64, error) {
	var (
		prs syscall.PtraceRegs
		err error
	)
	if prs, err = getRegisters(target.cmd); err != nil {
		return 0, err
	}
	return prs.Rbp, nil

}
