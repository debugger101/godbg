package main

import (
	"errors"
	"go.uber.org/zap"
	"golang.org/x/arch/x86/x86asm"
	"os"
	"path"
	"syscall"
)

type BInfo struct {
	original []byte
	filename string
	lineno int
	pc uint64
	kind BPKIND
}

type BP struct {
	infos []*BInfo
}

type BPKIND uint64

const (
	USERBPTYPE BPKIND = 1
	INTERNALBPTYPE BPKIND = 2
)

func (b *BPKIND)String() string{
	if *b == USERBPTYPE {
		return "USERBPTYPE"
	}
	if *b == INTERNALBPTYPE {
		return "INTERNALBPTYPE"
	}
	return "unknown"
}

func (bp *BP)setPcBreakPoint(pc uint64) ([]byte, error){
	// no need to add RwLock
	var err error
	if bp.infos == nil {
		bp.infos = make([]*BInfo, 0, 1)
	}
	for _, v := range bp.infos {
		if v.pc == pc {
			return nil, HasExistedBreakPointErr
		}
	}

	original := make([]byte, 1)
	_, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(pc), original)
	if err != nil {
		return nil, err
	}

	_, err = syscall.PtracePokeData(cmd.Process.Pid, uintptr(pc), []byte{0xCC})
	if err != nil {
		return nil, err
	}

	return original, nil
}

func (bp *BP)SetInternalBreakPoint(pc uint64) (*BInfo, error)  {
	var (
		original []byte
		err error
	)
	if original, err = bp.setPcBreakPoint(pc); err != nil {
		return nil, err
	}

	bInfo := &BInfo{original: original, filename: "", lineno: 0, pc: pc, kind: INTERNALBPTYPE}
	bp.infos = append(bp.infos, bInfo)
	return bInfo, nil
}

func (bp* BP)SetFileLineBreakPoint(filename string, lineno int) (*BInfo, error) {
	logger.Debug("SetFileLineBreakPoint", zap.String("filename", filename), zap.Int("lineno", lineno))
	curDir, err := os.Getwd()
	if err != nil {
		logger.Error("SetFileLineBreakPoint:GetWd", zap.Error(err), zap.Int(filename, lineno))
		return nil, err
	}

	fullfilename := path.Join(curDir, filename)
	pc, err := bi.fileLineToPc(fullfilename, lineno)
	if err != nil {
		logger.Error("SetFileLineBreakPoint:fileLineToPc",
			zap.Error(err),
			zap.Int(fullfilename, lineno))
		return nil, err
	}
	logger.Debug("SetFileLineBreakPoint:fileLineToPc",
		zap.Uint64("pc", pc),
		zap.String("fullfilename", fullfilename),
		zap.Int("lineno", lineno))

	var (
		info *BInfo
		original []byte
	)
	if original, err = bp.setPcBreakPoint(pc); err != nil{
		logger.Error("SetFileLineBreakPoint",
			zap.Error(err),
			zap.Int("Pid", cmd.Process.Pid),
			zap.String("fullfilename", fullfilename),
			zap.Int("lineno", lineno))
		return nil, err
	}
	info = &BInfo{original: original, filename: filename, lineno: lineno, pc: pc, kind: USERBPTYPE}
	bp.infos = append(bp.infos, info)

	return info, err
}

func (bp *BP)Continue() error {
	return syscall.PtraceCont(cmd.Process.Pid, 0)
}

func (bp *BP) findBreakPoint(pc uint64) (*BInfo , bool) {
	for _, v := range bp.infos {
		if v.pc == pc {
			return v, true
		}
	}
	return nil, false
}

func (bp *BP)enableBreakPoint (info *BInfo) error {
	if info == nil {
		return errors.New("enableBreakPoint breakpointinfo is null")
	}
	logger.Debug("enableBreakPoint", zap.Uint64("pc", info.pc))
	if _, err := syscall.PtracePokeData(cmd.Process.Pid, uintptr(info.pc), []byte{0xCC}); err != nil {
		return err
	}
	return nil
}

func (bp *BP)disableBreakPoint(info *BInfo) error {
	if info == nil {
		return errors.New("disableBreakPoint breakpointinfo is null")
	}
	logger.Debug("disableBreakPoint", zap.Uint64("pc", info.pc))
	if _, err := syscall.PtracePokeData(cmd.Process.Pid, uintptr(info.pc), info.original); err != nil {
		return err
	}
	return nil
}

func (bp *BP)getSingleMemInst(pc uint64) (x86asm.Inst, error){
	var (
		mem []byte
		err error
		inst x86asm.Inst
	)

	mem = make([]byte, 100)
	if _, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(pc), mem); err != nil {
		return x86asm.Inst{}, err
	}
	if inst ,err = x86asm.Decode(mem, 64); err != nil {
		return x86asm.Inst{}, err
	}
	return inst, nil
}

func (bp *BP)singleStepInstructionWithBreakpointCheck() (bool, error) {
	var (
		pc uint64
		err error
		info *BInfo
		ok bool

		inst x86asm.Inst
		interBpInfo *BInfo
	)
	if pc, err  = getPtracePc(); err != nil {
		return true, err
	}
	pc = pc - 1
	if info, ok = bp.findBreakPoint(pc); !ok {
		return true, nil
	}
	if err = bp.disableBreakPoint(info); err != nil {
		return true, err
	}
	defer bp.enableBreakPoint(info)
	if inst, err = bp.getSingleMemInst(pc); err != nil {
		return true, err
	}

	if interBpInfo, err = bp.SetInternalBreakPoint(pc + uint64(inst.Len)); err != nil {
		if  err != HasExistedBreakPointErr {
			return true, err
		}
		err = nil
	} else {
		defer func() {
			bp.disableBreakPoint(interBpInfo)
			bp.clearInternalBreakPoint(interBpInfo.pc)
		}()
	}

	if err = setPcRegister(pc); err != nil {
		return true, err
	}

	if err := syscall.PtraceCont(cmd.Process.Pid, 0); err != nil {
		return true, err
	}

	var s syscall.WaitStatus
	if _, err = syscall.Wait4(cmd.Process.Pid, &s, syscall.WALL, nil); err != nil {
		return true, err
	}
	status := (syscall.WaitStatus)(s)

	if status.Exited() {
		return true, nil
	}

	if pc, err  = getPtracePc(); err != nil {
		return true, err
	}

	if interBpInfo == nil || pc - 1 != interBpInfo.pc {
		return false, nil
	}

	return true, nil
}

func (bp *BP)clearInternalBreakPoint(pc uint64) {
	infos := make([]*BInfo, 0, len(bp.infos))
	for _, v := range bp.infos {
		if !(v.kind == INTERNALBPTYPE && v.pc == pc) {
			infos = append(infos, v)
		}
	}
	bp.infos = infos
}