package main

import (
	"fmt"
	"syscall"
    "golang.org/x/arch/x86/x86asm"
)

func disassemble(lowpc uint64, highpc uint64) ([]byte, error) {
	if highpc - lowpc <= 0 {
		return nil, fmt.Errorf("[disassemble] invalid input: lowpc %d highpc %d", lowpc, highpc)
	}
	mem := make([]byte, highpc - lowpc)

	var (
		n int
		err error
		asmInst x86asm.Inst
	)
	if n, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(lowpc), mem); err != nil {
		return nil, err
	}
	mem = mem[:n]

	var amsInsts []x86asm.Inst

	for len(mem) > 0 {
		if asmInst, err = x86asm.Decode(mem, 64); err != nil {
			return nil, err
		}
		amsInsts = append(amsInsts, asmInst)
		mem = mem[asmInst.Len:]

		fmt.Println(asmInst)
	}

	return nil, nil
}

// not considered inline function
func findFunctionIncludePc(pc uint64) (*Function, error) {
	for _, f := range bi.Functions {
		if f.lowpc <= pc && pc < f.highpc {
			return f, nil
		}
	}
	return nil, fmt.Errorf("[findFunctionIncludePc] can't find function by pc:%d", pc)
}

func disassembleByPtracePc() ([]byte, error) {
	pc, err := getPtracePc()
	if err != nil{
		return nil, err
	}
	f, err := findFunctionIncludePc(pc)
	if err != nil {
		return nil, err
	}

	disassemble(f.lowpc, f.highpc)

	return nil, nil
}
