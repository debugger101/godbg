package main

import (
	"debug/dwarf"
	"debug/elf"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
)

func analyze(execfile string) (*BI, error) {
	var (
		elffile *elf.File
		err error
		debugLineMapTableBytes []byte
		debugInfoBytes []byte
		dwarfData *dwarf.Data
		dwarfReader *dwarf.Reader
		curEntry *dwarf.Entry
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

	bi = &BI{Sources: make(map[string]map[int]uint64), Functions: make(map[string]uint64)}
	for {
		if curEntry, err = dwarfReader.Next(); err != nil{
			return nil, err
		}
		if curEntry == nil {
			break
		}

		if curEntry.Tag == dwarf.TagCompileUnit {
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
						bi.Sources[lineEntry.File.Name] = make(map[int]uint64)
					}

					if _, ok := bi.Sources[lineEntry.File.Name][lineEntry.Line]; !ok {
						bi.Sources[lineEntry.File.Name][lineEntry.Line] = lineEntry.Address
					}
				}
			}
		}
	}

	// debug log
	for file, mp := range bi.Sources {
		for line, addr := range mp {
			logger.Debug("bi",
				zap.String("file", file), zap.Int("line", line), zap.Uint64("addr", addr))
		}
	}

	_ = debugLineMapTableBytes
	_ = debugInfoBytes

	return bi, nil
}

type BI struct {
	Sources map[string]map[int]uint64
	Functions map[string]uint64
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
	if b.Sources[filename] == nil || b.Sources[filename][lineno] == 0 {
		return 0, NotFoundSourceLineErr
	}
	return b.Sources[filename][lineno], nil
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
		for lineno, addr := range filenameMp {
			if addr == pc {
				return filename, lineno, nil
			}
			if addr < pc && (!rangeMin.existedPc || addr > rangeMin.pc) {
				rangeMin.pc = addr
				rangeMin.existedPc = true
				rangeMin.filename = filename
				rangeMin.lineno = lineno
			}
			if pc < addr && (!rangeMax.existedPc || addr < rangeMax.pc) {
				rangeMax.pc = addr
				rangeMax.existedPc = true
				rangeMax.filename = filename
				rangeMax.lineno = lineno
			}
		}
	}

	if !(rangeMax.existedPc && rangeMax.existedPc) {
		return "", 0, errors.New("invalid input")
	}

	if (rangeMax.pc - pc) > (pc - rangeMin.pc) {
		return rangeMin.filename, rangeMin.lineno, nil
	}

	return rangeMax.filename, rangeMax.lineno, nil
}
