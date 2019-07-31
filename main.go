package main

import (
	"debug/dwarf"
	"debug/elf"
	"errors"
	"fmt"
	"github.com/chainhelen/godbg/log"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
)

func printHelper() {
	fmt.Printf("%s\n", "Usage:\n\tJust like ./godgb debug main.go")
}

func checkArgs() error {
	logger.Debug("[checkArgs]", zap.Strings("args", os.Args))
	if len(os.Args) != 3 {
		return errors.New("len(args) != 3")
	}
	debug := os.Args[1]

	if debug != "debug" {
		return errors.New("only support `debug`")
	}
	if  path.Ext(os.Args[2]) != ".go" {
		return errors.New("please input .go file")
	}
	return nil
}

func absoluteFilename() (string, error) {
	filename := os.Args[2]
	if path.IsAbs(filename) {
		return filename, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	filename = path.Join(dir, filename)
	_, err = os.Stat(filename)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func build (filename string) (string, error) {
	base := filepath.Base(filename)
	execfile := path.Join(os.TempDir(), "__" + base)

	args := []string{"build", "-gcflags", "all=-N -l", "-o", execfile, filename}

	cmd := exec.Command("go", args...)
	return execfile, cmd.Run()
}

// not supoort arguments of cmds
func runexec(execfile string) (*exec.Cmd, error){
	cmd := exec.Command(execfile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Ptrace: true, Setpgid: true, Foreground: false}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func analyze(execfile string) error {
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
	)
	if elffile, err = elf.Open(execfile); err != nil {
		return err
	}
	defer elffile.Close()

	lineSession := elffile.Section(".debug_line")
	if lineSession == nil {
		lineSession = elffile.Section(".zdebug_line")
	}
	if lineSession == nil {
		return errors.New("Can't not find .debug_line or .zdebug_line")
	}
	// please note that Data() returns uncompressed data if compressed
	if debugLineMapTableBytes, err = lineSession.Data(); err != nil{
		return err
	}


	infoSession := elffile.Section(".debug_info")
	if infoSession == nil {
		infoSession = elffile.Section(".zdebug_info")
	}
	if infoSession == nil {
		return errors.New("Can't not find .debug_info or .zdebug_info")
	}
	// please note that Data() returns uncompressed data if compressed
	if debugInfoBytes, err = infoSession.Data(); err != nil {
		return err
	}

	if dwarfData, err = elffile.DWARF(); err != nil {
		return err
	}
	dwarfReader = dwarfData.Reader()

	for {
		if curEntry, err = dwarfReader.Next(); err != nil{
			return err
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
			if ranges, err = dwarfData.Ranges(curEntry); err != nil {
				return err
			}
			_ = ranges
			/*
			if ranges != nil && len(ranges) >= 1{
				lowPc := ranges[0][0]
				hightPc := ranges[0][1]
			}
			*/


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
			}
		}
	}


	_ = debugLineMapTableBytes
	_ = debugInfoBytes

	return nil
}

var logger = log.Log

func main() {
	var (
		filename string
		execfile string
		cmd *exec.Cmd
		err      error
	)

	if err = checkArgs(); err != nil {
		logger.Error(err.Error(), zap.String("stage","checkArgs"), zap.Strings("args", os.Args))
		printHelper()
		return
	}

	// step 1, get absolute filename
	if filename, err = absoluteFilename(); err != nil {
		logger.Error(err.Error(), zap.String("stage","absolute"), zap.String("filename", filename))
		printHelper()
		return
	}

	// step 2, build the filename into executable file
	if execfile, err = build(filename); err != nil {
		logger.Error(err.Error(), zap.String("stage", "build"),zap.String("filename", filename))
		printHelper()
		return
	}
	defer os.Remove(execfile)

	// step 3, analyze executable file; The most import places are "_debug_info", "_debug_line"
	if err = analyze(execfile);err != nil {
		logger.Error(err.Error(), zap.String("stage", "analyze"),
			zap.String("filename", filename), zap.String("execfile", execfile))
		printHelper()
		return
	}

	// step 4, run executable file
	if cmd, err = runexec(execfile); err != nil {
		logger.Error(err.Error(), zap.String("stage", "runexec"),
			zap.String("filename", filename), zap.String("execfile", execfile))
		printHelper()
		return
	}
	_ = cmd
}
