package main

import (
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/arch/x86/x86asm"
	"os"
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
		curMem []byte
	)
	if n, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(lowpc), mem); err != nil {
		return nil, nil, nil, nil, err
	}
	mem = mem[:n]

	asmInsts := make([]x86asm.Inst, 0, len(mem))
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
		// Optimized display disassemble code where contain breakpoint
		if curbp, ok := bpMap[curPc]; ok {
			logger.Debug("disassemble", zap.Any("hit bp", curbp))
			copy(mem, curbp.original)
		}

		if asmInst, err = x86asm.Decode(mem, 64); err != nil {
			logger.Error("disassemble", zap.Error(err), zap.Uint64("pc", curPc), zap.ByteString("mem[0]", mem[:1]))
			return nil, nil, nil, nil, err
		}

		curMem = mem[:asmInst.Len]
		memSlice = append(memSlice, curMem)
		asmInsts = append(asmInsts, asmInst)
		pcs = append(pcs, curPc)
		logger.Debug("disassemble signle Inst",zap.Uint64("pc", curPc), zap.String("inst", asmInst.String()))

		mem = mem[asmInst.Len:]
		curPc += uint64(asmInst.Len)
	}

	return pcMap, memSlice, pcs, asmInsts, nil
}

// not considered inline function
func findFunctionIncludePc(pc uint64) (*Function, error) {
	for _, f := range bi.Functions {
		if f.lowpc <= pc && pc < f.highpc {
			return f, nil
		}
	}
	return nil, &NotFoundFuncErr{pc: pc}
}

func tryCuttingFilename(filename string) string {
	var (
		dir string
		err error
	)
	if dir , err = os.Getwd(); err != nil {
		return filename
	}
	dir += "/"
	if ok := strings.HasPrefix(filename, dir); ok {
		return filename[len(dir):]
	}
	return filename
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

	fmt.Fprintf(stdout,"current process pc = %d\n", pc)
	for i, amsInst := range amsInsts {
		curpc := pcs[i]
		if filename, lineno, err = bi.pcTofileLine(curpc); err != nil {
			return err
		}

		bpFlag := " "
		if pcBpMap[curpc] {
			bpFlag ="."
		}
		if i < len(pcs) - 1 && pcs[i] <= pc && pc < pcs[i + 1] {
			bpFlag += "===> "
		} else {
			bpFlag += "     "
		}

		out = append(out, fmt.Sprintf("%s%s:%-7d %-7d %-20x %s\n",bpFlag, path.Base(filename), lineno, curpc, mems[i], amsInst.String()))
	}
	fmt.Fprintln(stdout, strings.Join(out, ""))
	return nil
}