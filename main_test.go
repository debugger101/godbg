package main

import (
	"github.com/chainhelen/godbg/log"
	. "github.com/onsi/gomega"
	"os"
	"path"
	"strings"
	"testing"
)

func clear_variable() {
	bp = &BP{}
	logger = log.Log
	bi = nil
	cmd = nil

	stdin  = os.Stdin
	stdout = os.Stdout
	stderr = os.Stderr
}

func build_run_debug(filename string) (string, error) {
	var (
		dir string
		execfile string
		err      error
	)

	// step1, get absolute filename
	if dir, err = os.Getwd(); err != nil {
		return "", err
	}
	filename = path.Join(dir, filename)

	// step 2, build the filename into executable file
	if execfile, err = build(filename); err != nil {
		return "", err
	}

	// step 3, analyze executable file; The most import places are "_debug_info", "_debug_line"
	if bi, err = analyze(execfile); err != nil{
		return execfile, err
	}

	// step 4, run executable file
	if cmd, err = runexec(execfile); err != nil {
		return execfile, err
	}

	if err = os.Setenv("GODBG_TEST", "true");err != nil {
		return execfile, err
	}
	return execfile, nil
}

func make_out_err() (*strings.Builder,*strings.Builder){
	outWriter := &strings.Builder{}
	errWriter := &strings.Builder{}

	stdout = outWriter
	stderr = errWriter

	return outWriter, errWriter
}

func TestBuild(t *testing.T) {
	var (
		filename string
		execfile string
		err      error
	)
	g := NewGomegaWithT(t)
	dir, err := os.Getwd()
	g.Expect(err).Should(BeNil())
	filename = path.Join(dir, "./test_file/t1.go")

	execfile, err = build(filename)
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)

	clear_variable()
}


func TestQuit(t *testing.T) {
	var (
		execfile string
		err error
		g = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t1.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)

	executor("q")

	g.Expect(outw.Len()).Should(Equal(0))
	g.Expect(errw.Len()).Should(Equal(0))

	clear_variable()
}

func TestVarDefLineBreakPoint(t *testing.T) {
	var (
		execfile string
		err error
		g = NewGomegaWithT(t)
	)

	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t1.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)

	executor("b ./test_file/t1.go:6")

	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t1.go:6 breakpoint successfully"))
	g.Expect(errw.Len()).Should(Equal(0))

	executor("q")
	clear_variable()
}

func TestFuncDefLineBreakPoint(t *testing.T) {
	var (
		execfile string
		err error
		g = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t1.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)

	executor("b ./test_file/t1.go:5")

	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t1.go:5 breakpoint successfully"))
	g.Expect(errw.Len()).Should(Equal(0))

	executor("q")
	clear_variable()
}

func TestFuncCallLineBreakPoint(t *testing.T) {
	var (
		execfile string
		err error
		g = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t1.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)

	executor("b ./test_file/t1.go:12")

	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t1.go:12 breakpoint successfully"))
	g.Expect(errw.String()).Should(Equal(""))

	executor("q")
	clear_variable()
}

func TestVarDefLineContinue(t *testing.T) {
	var (
		execfile string
		err error
		g = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t1.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)
	pid := cmd.Process.Pid

	executor("b ./test_file/t1.go:6")
	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t1.go:6 breakpoint successfully"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring("==>      6: 	i := 20"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(Equal(""))
	g.Expect(errw.String()).Should(MatchRegexp("Process %d has exited with status 0", pid))

	executor("q")
	clear_variable()
}

func TestFuncDefLineContinue(t *testing.T) {
	var (
		execfile string
		err error
		g = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t1.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)
	pid := cmd.Process.Pid

	executor("b ./test_file/t1.go:5")
	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t1.go:5 breakpoint successfully"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring("==>      5: func p()"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(Equal(""))
	g.Expect(errw.String()).Should(MatchRegexp("Process %d has exited with status 0", pid))

	executor("q")
	clear_variable()
}

func TestFuncCallLineContinue(t *testing.T) {
	var (
		execfile string
		err error
		g = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t1.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)
	pid := cmd.Process.Pid

	executor("b ./test_file/t1.go:12")

	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t1.go:12 breakpoint successfully"))
	g.Expect(errw.String()).Should(Equal(""))

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring("==>     12: 	p()"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(Equal(""))
	g.Expect(errw.String()).Should(MatchRegexp("Process %d has exited with status 0", pid))

	executor("q")
	clear_variable()
}

func TestForExpressionContinue(t *testing.T) {
	var (
		execfile string
		err error
		g = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t2.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)
	pid := cmd.Process.Pid

	executor("b ./test_file/t2.go:7")

	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t2.go:7 breakpoint successfully"))
	g.Expect(errw.String()).Should(Equal(""))

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring("==>      7: 		fmt.Println(i)"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring("==>      7: 		fmt.Println(i)"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring("==>      7: 		fmt.Println(i)"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(Equal(""))
	g.Expect(errw.String()).Should(MatchRegexp("Process %d has exited with status 0", pid))

	executor("q")
	clear_variable()
}