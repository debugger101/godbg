package main

import (
	"go.uber.org/zap"
	"os"
	"path"
	"syscall"
)

type BInfo struct {
	original []byte
	filename string
	lineno int
	pc uint64
}

type BP struct {
	infos []*BInfo
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

	// no need to add RwLock
	if bp.infos == nil {
		bp.infos = make([]*BInfo, 0, 1)
	}
	for _, v := range bp.infos {
		if v.filename == fullfilename && v.lineno == lineno {
			logger.Error("SetFileLineBreakPoint",
				zap.Error(HasExistedBreakPointErr),
				zap.Int("Pid", cmd.Process.Pid),
				zap.String("fullfilename", fullfilename),
				zap.Int("lineno", lineno))
			return nil, HasExistedBreakPointErr
		}
	}

	original := make([]byte, 1)
	_, err = syscall.PtracePeekData(cmd.Process.Pid, uintptr(pc), original)
	if err != nil {
		logger.Error("SetFileLineBreakPoint:PtracePeekData",
			zap.Error(err),
			zap.Int("Pid", cmd.Process.Pid),
			zap.String("fullfilename", fullfilename),
			zap.Int("lineno", lineno))
		return nil, err
	}

	_, err = syscall.PtracePokeData(cmd.Process.Pid, uintptr(pc), []byte{0xCC})
	if err != nil {
		logger.Error("SetFileLineBreakPoint:PtracePokeData",
			zap.Error(err),
			zap.Int("Pid", cmd.Process.Pid),
			zap.String("fullfilename", fullfilename),
			zap.Int("lineno", lineno))
		return nil, err
	}

	bInfo := &BInfo{original: original, filename: fullfilename, lineno: lineno, pc: pc}
	bp.infos = append(bp.infos, bInfo)

	return bInfo, nil
}

func (bp *BP)Continue() error {
	return syscall.PtraceCont(cmd.Process.Pid, 0)
}
