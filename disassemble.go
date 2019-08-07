package main

import (
	"fmt"
	"path"
	"strings"
	"syscall"
    "golang.org/x/arch/x86/x86asm"
)

func disassemble(lowpc uint64, highpc uint64) ([][]byte, []uint64,[]x86asm.Inst, error) {
	if highpc - lowpc <= 0 {
		return nil, nil, nil, fmt.Errorf("[disassemble] invalid input: lowpc %d highpc %d", lowpc, highpc)
	}
	mem := make([]byte, highpc - lowpc)

	var (
		n int
		err error
		asmInst x86asm.Inst
	)
	if n, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(lowpc), mem); err != nil {
		return nil, nil, nil, err
	}
	mem = mem[:n]

	amsInsts := make([]x86asm.Inst, 0, len(mem))
	pcs := make([]uint64, 0, len(mem))
	memSlice := make([][]byte, 0, len(mem))

	curPc := lowpc
	for len(mem) > 0 {
		if asmInst, err = x86asm.Decode(mem, 64); err != nil {
			return nil, nil, nil, err
		}
		amsInsts = append(amsInsts, asmInst)
		pcs = append(pcs, curPc)
		memSlice = append(memSlice, mem[:asmInst.Len])

		mem = mem[asmInst.Len:]
		curPc += uint64(asmInst.Len)
	}

	return memSlice, pcs,amsInsts, nil
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

func listDisassembleByPtracePc() error {
	var (
		pc uint64
		pcs []uint64
		amsInsts []x86asm.Inst
		err error
		filename string
		lineno int
		f *Function
		mems [][]byte
	)

	if pc, err = getPtracePc(); err != nil {
		return err
	}
	if f, err = findFunctionIncludePc(pc); err != nil {
		return err
	}
	if mems, pcs, amsInsts, err = disassemble(f.lowpc, f.highpc); err != nil {
		return err
	}
	out := make([]string, 0, len(amsInsts))
	for i, amsInst := range amsInsts {
		curpc := pcs[i]
		if filename, lineno, err = bi.pcTofileLine(pc); err != nil {
			return err
		}
		if pc == curpc {
			out = append(out, fmt.Sprintf("==> %s:%-7d %-20x %s\n", path.Base(filename), lineno, mems[i], amsInst.String()))
		} else {
			out = append(out, fmt.Sprintf("    %s:%-7d %-20x %s\n", path.Base(filename), lineno, mems[i], amsInst.String()))
		}
	}
	fmt.Println(strings.Join(out, ""))
	return nil
}