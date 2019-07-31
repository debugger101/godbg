package main

import (
	"github.com/c-bata/go-prompt"
	"go.uber.org/zap"
	"os"
	"strings"
)

func executor(input string) {
	logger.Debug("executor", zap.String("input", input))

	input = strings.ToLower(input)
	if len(input) == 0 {
		return
	}
	fs := input[0]

	switch fs {
	case 'q':
		if input == "q" || input == "quit"{
			os.Exit(0)
		}
		printUnsupportCmd(input)
	case 'b':
		sps := strings.Split(input, " ")
		if len(sps) == 2 && (sps[0] == "b" || sps[0] == "break") {
			pc, err := bi.LineToPc(sps[1])
			if err != nil {
				printUnsupportCmd(input)
				return
			}
			_ = pc
			return
		}
		printUnsupportCmd(input)
	case 'c':
		printUnsupportCmd(input)
	}
	printUnsupportCmd(input)
}

func complete(docs prompt.Document) []prompt.Suggest {
	_ = docs
	return nil
}
