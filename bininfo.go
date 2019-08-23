package main

import (
	"debug/dwarf"
	"debug/elf"
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

	cu *CompileUnit
}

type BI struct {
	Sources map[string]map[int][]*dwarf.LineEntry
	Functions []*Function
	CompileUnits []*CompileUnit
}


func analyze(execfile string) (*BI, error) {
	var (
		elffile *elf.File
		err error
		debugLineMapTableBytes []byte
		debugInfoBytes []byte
		dwarfData *dwarf.Data
		dwarfReader *dwarf.Reader
		curEntry *dwarf.Entry
		curSubProgramEntry *dwarf.Entry
		curCompileUnitEntry *dwarf.Entry
		curCompileUnit *CompileUnit
		curFunction *Function
		ranges [][2]uint64
		lineReader *dwarf.LineReader
		lineEntry *dwarf.LineEntry
		bi *BI
	)
	if elffile, err = elf.Open(execfile); err != nil {
		return nil, err
	}
	defer elffile.Close()

	lineSession := elffile.Section(".debug_line")
	if lineSession == nil {
		lineSession = elffile.Section(".zdebug_line")
	}
	if lineSession == nil {
		return nil, errors.New("Can't not find .debug_line or .zdebug_line")
	}
	// please note that Data() returns uncompressed data if compressed
	if debugLineMapTableBytes, err = lineSession.Data(); err != nil{
		return nil, err
	}


	infoSession := elffile.Section(".debug_info")
	if infoSession == nil {
		infoSession = elffile.Section(".zdebug_info")
	}
	if infoSession == nil {
		return nil, errors.New("Can't not find .debug_info or .zdebug_info")
	}
	// please note that Data() returns uncompressed data if compressed
	if debugInfoBytes, err = infoSession.Data(); err != nil {
		return nil, err
	}

	if dwarfData, err = elffile.DWARF(); err != nil {
		return nil, err
	}
	dwarfReader = dwarfData.Reader()

	bi = &BI{Sources: make(map[string]map[int][]*dwarf.LineEntry)}
	for {
		if curEntry, err = dwarfReader.Next(); err != nil{
			return nil, err
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
				return nil, err
			}
			lineEntry = &dwarf.LineEntry{}
			cuname, _ := curEntry.Val(dwarf.AttrName).(string)
			for {
				if err = lineReader.Next(lineEntry); err != nil && err != io.EOF{
					return nil, err
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
	}

	// debug source log
	for file, mp := range bi.Sources {
		for line, lineEntryArray := range mp {
			for _, lineEntry := range lineEntryArray {
				logger.Debug("bi",
					zap.String("file", file), zap.Int("line", line), zap.Uint64("addr", lineEntry.Address))
			}
		}
	}

	_ = debugLineMapTableBytes
	_ = debugInfoBytes
	_ = curSubProgramEntry
	_ = curCompileUnitEntry

	return bi, nil
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