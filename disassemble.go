package main

import (
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/arch/x86/x86asm"
	"path"
	"strings"
	"syscall"
)

func disassemble(lowpc uint64, highpc uint64) (map[uint64]bool, [][]byte, []uint64,[]x86asm.Inst, error) {
	if highpc - lowpc <= 0 {
		return nil, nil, nil, nil, fmt.Errorf("[disassemble] invalid input: lowpc %d highpc %d", lowpc, highpc)
	}
	mem := make([]byte, highpc - lowpc)

	var (
		n int
		err error
		asmInst x86asm.Inst
		bpMap map[uint64]*BInfo
		pcMap map[uint64]bool
	)
	if n, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(lowpc), mem); err != nil {
		return nil, nil, nil, nil, err
	}
	mem = mem[:n]

	amsInsts := make([]x86asm.Inst, 0, len(mem))
	pcs := make([]uint64, 0, len(mem))
	memSlice := make([][]byte, 0, len(mem))

	// Optimized display disassemble code where contain breakpoint
	bpMap = make(map[uint64]*BInfo, len(bp.infos))
	pcMap = make(map[uint64]bool, len(bp.infos))
	for _, info := range bp.infos {
		bpMap[info.pc] = info
		pcMap[info.pc] = true
	}

	curPc := lowpc
	for len(mem) > 0 {
		if asmInst, err = x86asm.Decode(mem, 64); err != nil {
			logger.Error("disassemble", zap.Error(err), zap.Uint64("pc", curPc))
			return nil, nil, nil, nil, err
		}

		// Optimized display disassemble code where contain breakpoint
		if bpMap[curPc] != nil {
			curbp := bpMap[curPc]
			if asmInst, err = x86asm.Decode(curbp.original, 64);err != nil {
				logger.Error("disassemble", zap.Error(err), zap.Uint64("pc", curPc))
				return nil, nil, nil, nil, err
			}
			memSlice = append(memSlice, curbp.original)
		} else {
			memSlice = append(memSlice, mem[:asmInst.Len])
		}
		amsInsts = append(amsInsts, asmInst)
		pcs = append(pcs, curPc)

		mem = mem[asmInst.Len:]
		curPc += uint64(asmInst.Len)
	}

	return pcMap, memSlice, pcs, amsInsts, nil
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
		pcBpMap map[uint64]bool
	)

	if pc, err = getPtracePc(); err != nil {
		return err
	}
	if f, err = findFunctionIncludePc(pc); err != nil {
		return err
	}
	if pcBpMap, mems, pcs, amsInsts, err = disassemble(f.lowpc, f.highpc); err != nil {
		return err
	}
	out := make([]string, 0, len(amsInsts))


	for i, amsInst := range amsInsts {
		curpc := pcs[i]
		if filename, lineno, err = bi.pcTofileLine(pc); err != nil {
			return err
		}

		bpFlag := " "
		if pcBpMap[curpc] {
			bpFlag ="."
		}

		if pc == curpc {
			out = append(out, fmt.Sprintf("%s==> %s:%-7d %-7d %-20x %s\n",bpFlag, path.Base(filename), lineno, curpc, mems[i], amsInst.String()))

		} else {
			out = append(out, fmt.Sprintf("%s    %s:%-7d %-7d %-20x %s\n",bpFlag, path.Base(filename), lineno, curpc, mems[i], amsInst.String()))
		}
	}
	fmt.Println(strings.Join(out, ""))
	return nil
}