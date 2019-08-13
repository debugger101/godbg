package main

import (
	"bufio"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"strings"
)

func listFileLineByPtracePc(rangeline int) error {
	pc, err := getPtracePc()
	if err != nil{
		return err
	}
	filename, lineno, err := bi.pcTofileLine(pc)
	if err != nil {
		return err
	}
	logger.Debug("list", zap.Int(filename, lineno))
	return listFileLine(filename, lineno, rangeline)
}

func listFileLine(filename string, lineno int, rangeline int) error{
	rangeMin := lineno - rangeline - 1
	rangeMax := lineno + rangeline - 1

	if rangeMin < 1 {
		rangeMin = 1
	}

	if rangeMax - rangeMin <= 0 {
		return errors.New("not right linenoe or rangeline")
	}

	file, err := os.OpenFile(filename, os.O_RDONLY, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	listFileLineBytesSlice := make([]string, 0, rangeMax - rangeMin + 2)

	listFileLineBytesSlice = append(listFileLineBytesSlice, fmt.Sprintf("list %s:%d\n", filename, lineno))
	var curLine int
	for {
		curLine++
		if curLine > rangeMax {
			break
		}

		lineBytes, err := reader.ReadSlice('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if rangeMin <= curLine && curLine <= rangeMax {
			if curLine == lineno {
				lineBytes = append([]byte(fmt.Sprintf("==>%7d: ", curLine)), lineBytes...)
			} else {
				lineBytes = append([]byte(fmt.Sprintf("   %7d: ", curLine)), lineBytes...)
			}
			listFileLineBytesSlice = append(listFileLineBytesSlice, string(lineBytes))
		}
	}

	fmt.Fprintln(stdout, strings.Join(listFileLineBytesSlice, ""))

	return nil
}
