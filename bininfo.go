package main

import (
	"bytes"
	"compress/zlib"
	"debug/dwarf"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/arch/x86/x86asm"
	"io"
	"strconv"
	"strings"
	"syscall"
)

type CompileUnit struct {
	functions []*Function
}

type Function struct {
	name string
	lowpc uint64
	highpc uint64
	frameBase []byte
	declFile int64
	external bool

	variables []*dwarf.Entry
	cu *CompileUnit
}

type BI struct {
	Sources map[string]map[int][]*dwarf.LineEntry
	Functions []*Function
	CompileUnits []*CompileUnit
	FramesInformation []*VirtualUnwindFrameInformation
}


func analyze(execfile string) (*BI, error) {
	var (
		elffile *elf.File
		err error
		dwarfData *dwarf.Data
		bi *BI
	)
	if elffile, err = elf.Open(execfile); err != nil {
		return nil, err
	}
	defer elffile.Close()

	// just check
	if _, err = openInfoSection(elffile); err != nil {
		return nil, err
	}

	if _, err = openLineSection(elffile); err != nil {
		return nil, err
	}

	// parse
	bi = &BI{Sources: make(map[string]map[int][]*dwarf.LineEntry)}
	if dwarfData, err = elffile.DWARF(); err != nil {
		return nil, err
	}
	if err = bi.ParseLineAndInfoSection(dwarfData); err != nil {
		return nil, err
	}
	if err = bi.ParseFrameSection(elffile); err != nil {
		return nil, err
	}

	// debug source log
	for file, mp := range bi.Sources {
		for line, lineEntryArray := range mp {
			for _, lineEntry := range lineEntryArray {
				logger.Debug("bi.sources",
					zap.String("file", file), zap.Int("line", line), zap.Uint64("addr", lineEntry.Address))
			}
		}
	}
	// debug frame log
	for i, v := range bi.FramesInformation {
		if v.CIE != nil {
			logger.Debug("bi.frames", zap.Int("index", i), zap.String("cie", v.CIE.String()))
		} else if v.FDE != nil {
			logger.Debug("bi.frames", zap.Int("index", i), zap.String("fde", v.FDE.String()))
		} else {
			logger.Error("find frame both cie/pde == nil")
		}
	}

	return bi, nil
}

func openInfoSection(elffile *elf.File) ([]byte, error) {
	var (
		debugInfoBytes []byte
		err error
	)
	infoSection := elffile.Section(".debug_info")
	if infoSection == nil {
		infoSection = elffile.Section(".zdebug_info")
	}
	if infoSection == nil {
		return nil, errors.New("Can't not find .debug_info or .zdebug_info")
	}
	// please note that Data() returns uncompressed data if compressed
	if debugInfoBytes, err = infoSection.Data(); err != nil {
		return nil, err
	}
	return debugInfoBytes, nil
}

func openLineSection(elffile *elf.File)([]byte, error) {
	var (
		debugLineMapTableBytes []byte
		err error
	)
	lineSection := elffile.Section(".debug_line")
	if lineSection == nil {
		lineSection = elffile.Section(".zdebug_line")
	}
	if lineSection == nil {
		return nil, errors.New("Can't not find .debug_line or .zdebug_line")
	}
	// please note that Data() returns uncompressed data if compressed
	if debugLineMapTableBytes, err = lineSection.Data(); err != nil{
		return nil, err
	}
	return debugLineMapTableBytes, nil
}

func (bi *BI)ParseLineAndInfoSection(dwarfData *dwarf.Data) error {
	var (
		curEntry *dwarf.Entry
		curCompileUnit *CompileUnit
		curFunction *Function
		err error
		ranges [][2]uint64
		lineReader *dwarf.LineReader
		lineEntry *dwarf.LineEntry
		curSubProgramEntry *dwarf.Entry
		curCompileUnitEntry *dwarf.Entry
		dwarfReader *dwarf.Reader
	)
	dwarfReader = dwarfData.Reader()
	for {
		if curEntry, err = dwarfReader.Next(); err != nil{
			return err
		}
		if curEntry == nil {
			break
		}


		if curEntry.Tag == dwarf.TagCompileUnit {
			curCompileUnit = &CompileUnit{}
			bi.CompileUnits = append(bi.CompileUnits, curCompileUnit)

			fields := curEntry.Field
			logger.Debug("|================= START ===========================|")
			for _, field := range fields {
				// for debug log
				logger.Debug("TagCompileUnit",
					zap.String("Attr", field.Attr.String()),
					zap.String("Val", fmt.Sprintf("%v", field.Val)),
					zap.String("Class", fmt.Sprintf("%s", field.Class)))
			}
			logger.Debug("|================== END ============================|")

			// LowPc(Attr) + Ranges(Attr) = HighPc, (* Data)Ranges return [LowPc, HightPc]
			/*if ranges, err = dwarfData.Ranges(curEntry); err != nil {
				return nil, err
			}

			if ranges != nil && len(ranges) >= 1{
				lowPc := ranges[0][0]
				hightPc := ranges[0][1]
			}
			*/
			_ = ranges


			if lineReader, err = dwarfData.LineReader(curEntry); err != nil {
				return err
			}
			lineEntry = &dwarf.LineEntry{}
			cuname, _ := curEntry.Val(dwarf.AttrName).(string)
			for {
				if err = lineReader.Next(lineEntry); err != nil && err != io.EOF{
					return err
				}
				if err == io.EOF {
					err = nil
					break
				}
				logger.Debug("cu:" + cuname, zap.Any("lineEntry", lineEntry))
				if lineEntry.File != nil {
					if bi.Sources[lineEntry.File.Name] == nil {
						bi.Sources[lineEntry.File.Name] = make(map[int][]*dwarf.LineEntry)
					}
					copyLineEntry := &dwarf.LineEntry{}
					*copyLineEntry = *lineEntry
					bi.Sources[lineEntry.File.Name][lineEntry.Line] = append(bi.Sources[lineEntry.File.Name][lineEntry.Line], copyLineEntry)
				}
			}

			curCompileUnitEntry = curEntry
		}

		if curEntry.Tag == dwarf.TagSubprogram {
			curFunction = &Function{}
			curCompileUnit.functions = append(curCompileUnit.functions, curFunction)
			curFunction.cu = curCompileUnit
			bi.Functions = append(bi.Functions, curFunction)

			fields := curEntry.Field
			logger.Debug("|================= START ===========================|")
			for _, field := range fields {
				switch field.Attr {
				case dwarf.AttrName:
					if val, ok := field.Val.(string); ok {
						curFunction.name = val
					}
				case dwarf.AttrLowpc:
					if val, ok := field.Val.(uint64); ok {
						curFunction.lowpc = val
					}
				case dwarf.AttrHighpc:
					if val, ok := field.Val.(uint64); ok {
						curFunction.highpc = val
					}
				case dwarf.AttrFrameBase:
					if val, ok := field.Val.([]byte); ok {
						curFunction.frameBase = val
					}
				case dwarf.AttrDeclFile:
					if val, ok := field.Val.(int64); ok {
						curFunction.declFile = val
					}
				case dwarf.AttrExternal:
					if val, ok := field.Val.(bool); ok {
						curFunction.external = val
					}
				default:
					logger.Debug("analyze:TagSubprogram unknow attr", zap.Any("field",field))
				}
				// for debug log
				logger.Debug("TagSubprogram",
					zap.String("Attr", field.Attr.String()),
					zap.String("Val", fmt.Sprintf("%v", field.Val)),
					zap.String("Class", fmt.Sprintf("%s", field.Class)))
			}
			logger.Debug("|================== END ============================|")

			curSubProgramEntry = curEntry
		}

		/*curEntry.Tag == dwarf.TagArrayType ||
		curEntry.Tag == dwarf.TagBaseType ||
		curEntry.Tag == dwarf.TagClassType ||
		curEntry.Tag == dwarf.TagStructType ||
		curEntry.Tag == dwarf.TagConstType ||
		curEntry.Tag == dwarf.TagPointerType ||
		curEntry.Tag == dwarf.TagStringType */
		if	curEntry.Tag == dwarf.TagVariable {
			curFunction.variables = append(curFunction.variables, curEntry)
			logger.Debug("|================= START ===========================|")
			fields := curEntry.Field
			for _, field := range fields {
				logger.Debug(curEntry.Tag.GoString(),
					zap.String("Attr", field.Attr.String()),
					zap.String("Val", fmt.Sprintf("%v", field.Val)),
					zap.String("Class", fmt.Sprintf("%s", field.Class)))
			}
			logger.Debug("|================== END ============================|")
		}
	}

	_ = curSubProgramEntry
	_ = curCompileUnitEntry
	return nil
}

// not considered inline function
func (bi *BI)findFunctionIncludePc(pc uint64) (*Function, error) {
	for _, f := range bi.Functions {
		if f.lowpc <= pc && pc < f.highpc {
			return f, nil
		}
	}
	return nil, &NotFoundFuncErr{pc: pc}
}

func (bi *BI)ParseFrameSection(elffile *elf.File) error {
	var (
		err error
		frameSection *elf.Section
		frameData []byte
		frameInfo *VirtualUnwindFrameInformation
	)
	frameSection = elffile.Section(".debug_frame")
	if frameSection == nil {
		frameSection = elffile.Section(".zdebug_frame")
		sectionData := func(s *elf.Section) ([]byte, error) {
			b, err := s.Data()
			if err != nil && uint64(len(b)) < s.Size {
				return nil, err
			}

			if len(b) >= 12 && string(b[:4]) == "ZLIB" {
				dlen := binary.BigEndian.Uint64(b[4:12])
				dbuf := make([]byte, dlen)
				r, err := zlib.NewReader(bytes.NewBuffer(b[12:]))
				if err != nil {
					return nil, err
				}
				if _, err := io.ReadFull(r, dbuf); err != nil {
					return nil, err
				}
				if err := r.Close(); err != nil {
					return nil, err
				}
				b = dbuf
			}
			return b, nil
		}
		if frameData, err = sectionData(frameSection); err != nil {
			return err
		}
	} else {
		if frameData, err = frameSection.Data(); err != nil {
			return err
		}
	}
	if frameSection == nil {
		return errors.New("can'tt find the .debug_frame or .zdebug_frame")
	}

	buffer := bytes.NewBuffer(frameData)
	var curCIE *CommonInformationEntry
	for {
		if frameInfo, err = parseFrameInformation(buffer); err != nil {
			if err == io.EOF {
				err = nil
				break
			}
		}
		if bi.FramesInformation == nil {
			bi.FramesInformation = make([]*VirtualUnwindFrameInformation, 0, 1)
		}
		bi.FramesInformation = append(bi.FramesInformation, frameInfo)
		if frameInfo.CIE != nil {
			curCIE = frameInfo.CIE
		}
		if frameInfo.FDE != nil {
			frameInfo.FDE.CIE = curCIE
		}
	}
	return err
}

func (bi *BI) findFrameInformation (pc uint64) (*Frame, error) {
	var fde *FrameDescriptionEntry
	for index, frameInfo := range bi.FramesInformation {
		if frameInfo.FDE != nil {
			if frameInfo.FDE.begin <= pc && pc <= (frameInfo.FDE.begin + frameInfo.FDE.size) {
				//fde = frameInfo.FDE
				//break
				if fde == nil {
					fde = frameInfo.FDE

					logger.Debug("findFrameInfomation", zap.Int("index", index))
				} else {
					return nil, fmt.Errorf("dumplicate fde")
				}
			}
		}
	}
	if fde == nil {
		return nil, fmt.Errorf("not find the frame cover pc = %d", pc)
	}

	cie := fde.CIE
	if cie == nil {
		return nil, fmt.Errorf("fde.CIE should not be nil")
	}

	frame := &Frame{cie: cie, cfa : &DWRule{}, regsRule: make(map[uint64]DWRule)}
	logger.Debug("========================= cie start\n")
	if err := execCIEInstructions(frame, bytes.NewBuffer(cie.initial_instructions)); err != nil {
		return nil, err
	}
	frame.loc = fde.begin
	frame.address = pc
	logger.Debug("========================= cie end\n")

	logger.Debug("========================= fde.instructions start \n")
	if err := execFDEInstructions(frame, bytes.NewBuffer(fde.instructions)); err != nil {
		return nil, err
	}
	logger.Debug("========================= fde.instructions end \n")

	var (
		regs syscall.PtraceRegs
		err error
	)
	if regs, err  = getRegisters(); err != nil {
		return nil, err
	}

	frame.regs = make([]uint64, 17)
	frame.regs[16] = regs.PC()
	frame.regs[7] = regs.Rsp
	frame.regs[6] = regs.Rbp

	logger.Debug("findFrameInformation",
		zap.Any("regs", regs),
		zap.Uint64("16", frame.regs[16]),
		zap.Uint64("07", frame.regs[7]),
		zap.Uint64("06", frame.regs[6]),
	)

	var framebase uint64
	switch frame.cfa.rule {
	case RuleCFA:
		if frame.cfa.reg >= 17 {
			return nil, fmt.Errorf("frame.cfa.reg >= 17")
		}
		if frame.regs[frame.cfa.reg] == 0 {
			return nil, fmt.Errorf("rule.Reg is null")
		}
		reg := frame.regs[frame.cfa.reg]
		framebase = reg + uint64(frame.cfa.offset)

		logger.Debug("findFrameInformation",
			zap.Uint64("frame.frambebase", frame.framebase),
			zap.Int64("offset", frame.cfa.offset),
			zap.Uint64("framebase", framebase))
	/*case RuleOffset:
		addr := frame.cfa.offset
		buf := make([]byte, 8)
		if _ ,err := syscall.PtracePeekData(cmd.Process.Pid, uintptr(addr), buf); err !=nil{
			return nil, err
		}
		v := binary.LittleEndian.Uint64(buf)
		frame.regs[7] = v
		framebase = v*/

	default:
		return nil, fmt.Errorf("invalid cfa rule %v", frame.cfa.rule)
	}

	frame.framebase = framebase

	return frame, nil
}

func parseLoc(loc string) (string, int, error) {
	sps := strings.Split(loc, ":")
	if len(sps) != 2{
		return "", 0, errors.New("wrong loc should be like filename:lineno")
	}
	filename, linenostr := sps[0], sps[1]
	lineno, err := strconv.Atoi(linenostr)
	if err != nil {
		return "", 0, errors.New("wrong loc should be like filename:lineno")
	}
	return filename, lineno, nil
}

func (b *BI) locToPc(loc string) (uint64, error){
	filename, lineno, err := parseLoc(loc)
	if err != nil {
		return 0, err
	}
	return b.fileLineToPc(filename, lineno)
}

func (b *BI) fileLineToPc(filename string, lineno int) (uint64, error) {
	if b.Sources[filename] == nil || b.Sources[filename][lineno] == nil || len(b.Sources[filename][lineno]) == 0{
		return 0, NotFoundSourceLineErr
	}
	return b.Sources[filename][lineno][0].Address, nil
}

func (b *BI) fileLineToPcForBreakPoint(filename string, lineno int) (uint64, error) {
	if b.Sources[filename] == nil || b.Sources[filename][lineno] == nil || len(b.Sources[filename][lineno]) == 0{
		return 0, NotFoundSourceLineErr
	}
	lineEntryArray := b.Sources[filename][lineno]
	for _, v := range lineEntryArray {
		if v.PrologueEnd {
			return v.Address, nil
		}
	}
	addr := uint64(0)
	for i, v := range lineEntryArray {
		if i == 0 {
			addr = v.Address
		} else {
			if addr > v.Address {
				addr = v.Address
			}
		}
	}
	if addr == 0 {
		return 0, NotFoundSourceLineErr
	}
	return addr, nil
}

func (b *BI) getCurFileLineByPtracePc() (string, int, error ){
	var (
		pc uint64
		err error
	)
	if pc, err = getPtracePc(); err != nil {
		printErr(err)
		return "", 0, errors.New("get ptrace pc failed")
	}
	return bi.pcTofileLine(pc)
}

func (b *BI) pcTofileLine(pc uint64)(string, int, error) {
	if b.Sources == nil {
		return "", 0, errors.New("no sources file")
	}

	type Rs struct {
		pc uint64
		existedPc bool
		filename string
		lineno int
	}

	rangeMin := &Rs{}
	rangeMax := &Rs{}


	for filename, filenameMp := range b.Sources {
		for lineno, lineEntryArray := range filenameMp {
			for _, lineEntry := range lineEntryArray {
				if lineEntry.Address == pc {
					return filename, lineno, nil
				}
				if lineEntry.Address <= pc && (!rangeMin.existedPc || lineEntry.Address > rangeMin.pc) {
					rangeMin.pc = lineEntry.Address
					rangeMin.existedPc = true
					rangeMin.filename = filename
					rangeMin.lineno = lineno
				}
				if pc < lineEntry.Address && (!rangeMax.existedPc || lineEntry.Address < rangeMax.pc) {
					rangeMax.pc = lineEntry.Address
					rangeMax.existedPc = true
					rangeMax.filename = filename
					rangeMax.lineno = lineno
				}
			}
		}
	}

	return rangeMin.filename, rangeMin.lineno, nil
}

func (bi *BI)getSingleMemInst(pc uint64) (x86asm.Inst, error){
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