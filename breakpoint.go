package main

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path"
	"syscall"
)

type BInfo struct {
	original []byte
	filename string
	lineno   int
	pc       uint64
	kind     BPKIND
}

type BP struct {
	infos []*BInfo
	pid   int
}

type BPKIND uint64

const (
	USERBPTYPE     BPKIND = 1
	INTERNALBPTYPE BPKIND = 2
)

func (b *BPKIND) String() string {
	if *b == USERBPTYPE {
		return "USERBPTYPE"
	}
	if *b == INTERNALBPTYPE {
		return "INTERNALBPTYPE"
	}
	return "unknown"
}

func (bp *BP) setPcBreakPoint(pid int, pc uint64) ([]byte, error) {
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
	_, err = syscall.PtracePeekData(pid, uintptr(pc), original)
	if err != nil {
		return nil, err
	}

	_, err = syscall.PtracePokeData(pid, uintptr(pc), []byte{0xCC})
	if err != nil {
		return nil, err
	}

	return original, nil
}

func (bp *BP) SetInternalBreakPoint(pid int, pc uint64) (*BInfo, error) {
	var (
		original []byte
		err      error
	)
	if original, err = bp.setPcBreakPoint(pid, pc); err != nil {
		return nil, err
	}

	bInfo := &BInfo{original: original, filename: "", lineno: 0, pc: pc, kind: INTERNALBPTYPE}
	bp.infos = append(bp.infos, bInfo)
	return bInfo, nil
}

func (bp *BP) SetFileLineBreakPoint(bi *BI, pid int, filename string, lineno int) (*BInfo, error) {
	logger.Debug("SetFileLineBreakPoint", zap.String("filename", filename), zap.Int("lineno", lineno))
	curDir, err := os.Getwd()
	if err != nil {
		logger.Error("SetFileLineBreakPoint:GetWd", zap.Error(err), zap.Int(filename, lineno))
		return nil, err
	}

	fullfilename := path.Join(curDir, filename)
	pc, err := bi.fileLineToPcForBreakPoint(fullfilename, lineno)
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
		info     *BInfo
		original []byte
	)
	if original, err = bp.setPcBreakPoint(pid, pc); err != nil {
		logger.Error("SetFileLineBreakPoint",
			zap.Error(err),
			zap.Int("Pid", pid),
			zap.String("fullfilename", fullfilename),
			zap.Int("lineno", lineno))
		return nil, err
	}
	info = &BInfo{original: original, filename: filename, lineno: lineno, pc: pc, kind: USERBPTYPE}
	bp.infos = append(bp.infos, info)

	return info, err
}

func (bp *BP) Continue(pid int) error {
	return syscall.PtraceCont(pid, 0)
}

func (bp *BP) findBreakPoint(pc uint64) (*BInfo, bool) {
	for _, v := range bp.infos {
		if v.pc == pc {
			return v, true
		}
	}
	return nil, false
}

func (bp *BP) enableBreakPoint(pid int, info *BInfo) error {
	if info == nil {
		return errors.New("enableBreakPoint breakpointinfo is null")
	}
	logger.Debug("enableBreakPoint", zap.Uint64("pc", info.pc))
	if _, err := syscall.PtracePokeData(pid, uintptr(info.pc), []byte{0xCC}); err != nil {
		return err
	}
	return nil
}

func (bp *BP) disableBreakPoint(pid int, info *BInfo) error {
	if info == nil {
		return errors.New("disableBreakPoint breakpointinfo is null")
	}
	logger.Debug("disableBreakPoint", zap.Uint64("pc", info.pc))
	if _, err := syscall.PtracePokeData(pid, uintptr(info.pc), info.original); err != nil {
		return err
	}
	return nil
}

func (bp *BP) singleStepInstructionWithBreakpointCheck(pid int) error {
	var (
		pc   uint64
		err  error
		info *BInfo
		ok   bool
	)

	if pc, err = getPtracePc(); err != nil {
		return err
	}
	pc = pc - 1
	if info, ok = bp.findBreakPoint(pc); !ok {
		return nil
	}
	if err = bp.disableBreakPoint(pid, info); err != nil {
		return err
	}
	defer bp.enableBreakPoint(pid, info)

	if err = setPcRegister(target.cmd, pc); err != nil {
		return err
	}

	if err = syscall.PtraceSingleStep(pid); err != nil {
		return err
	}
	var s syscall.WaitStatus
	if _, err = syscall.Wait4(pid, &s, syscall.WALL, nil); err != nil {
		return err
	}
	if s.Exited() {
		return nil
	}
	if s.StopSignal() == syscall.SIGTRAP {
		return nil
	}

	return fmt.Errorf("unknown waitstatus %v, signal %d", s, s.Signal())
}

func (bp *BP) clearInternalBreakPoint(pc uint64) {
	infos := make([]*BInfo, 0, len(bp.infos))
	for _, v := range bp.infos {
		if !(v.kind == INTERNALBPTYPE && v.pc == pc) {
			infos = append(infos, v)
		}
	}
	bp.infos = infos
}

func (bp *BP) SetBpWhenRestart(pid int) error {
	for _, v := range bp.infos {
		if v.kind == INTERNALBPTYPE {
			bp.clearInternalBreakPoint(v.pc)
		}
		if v.kind == USERBPTYPE {
			if err := bp.enableBreakPoint(pid, v); err != nil {
				return err
			}
		}
	}
	return nil
}
