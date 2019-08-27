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

	stdin = os.Stdin
	stdout = os.Stdout
	stderr = os.Stderr
}

func build_run_debug(filename string) (string, error) {
	var (
		dir      string
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
	if bi, err = analyze(execfile); err != nil {
		return execfile, err
	}

	// step 4, run executable file
	if cmd, err = runexec(execfile); err != nil {
		return execfile, err
	}

	if err = os.Setenv("GODBG_TEST", "true"); err != nil {
		return execfile, err
	}
	return execfile, nil
}

func make_out_err() (*strings.Builder, *strings.Builder) {
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
		err      error
		g        = NewGomegaWithT(t)
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
		err      error
		g        = NewGomegaWithT(t)
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
		err      error
		g        = NewGomegaWithT(t)
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
		err      error
		g        = NewGomegaWithT(t)
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
		err      error
		g        = NewGomegaWithT(t)
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
		err      error
		g        = NewGomegaWithT(t)
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
		err      error
		g        = NewGomegaWithT(t)
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
		err      error
		g        = NewGomegaWithT(t)
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

func TestBreakClearContinue(t *testing.T) {
	var (
		execfile string
		err      error
		g        = NewGomegaWithT(t)
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

	executor("bc 1")
	g.Expect(outw.String()).Should(ContainSubstring("clear breakpoint 1 successfully, resort breakpoint again"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("bl")
	g.Expect(outw.String()).Should(ContainSubstring("there is no breakpoint"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(Equal(""))
	g.Expect(errw.String()).Should(MatchRegexp("Process %d has exited with status 0", pid))

	executor("q")
	clear_variable()
}

func TestNextAssignExpression(t *testing.T) {
	var (
		execfile string
		err      error
		g        = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t3.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)
	pid := cmd.Process.Pid

	executor("b ./test_file/t3.go:6")

	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t3.go:6 breakpoint successfully"))
	g.Expect(errw.String()).Should(Equal(""))

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring("==>      6: 	m := 0"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("n")
	g.Expect(outw.String()).Should(ContainSubstring("==>      7: 	n := 1"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("n")
	g.Expect(outw.String()).Should(ContainSubstring("==>      8: 	i := 10"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("n")
	g.Expect(outw.String()).Should(ContainSubstring("==>      9: 	j := 11"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(Equal(""))
	g.Expect(errw.String()).Should(MatchRegexp("Process %d has exited with status 0", pid))

	executor("q")
	clear_variable()
}

func TestNextCallExpress(t *testing.T) {
	var (
		execfile string
		err      error
		g        = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t3.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)
	pid := cmd.Process.Pid

	executor("b ./test_file/t3.go:10")

	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t3.go:10 breakpoint successfully"))
	g.Expect(errw.String()).Should(Equal(""))

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring(`==>     10: 	fmt.Printf("%d %d %d %d\n", m, n, i, j)`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("n")
	g.Expect(outw.String()).Should(ContainSubstring(`==>     11: 	k := 20`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("n")
	g.Expect(outw.String()).Should(ContainSubstring(`==>     12: 	fmt.Printf("%d\n", k)`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(Equal(""))
	g.Expect(errw.String()).Should(MatchRegexp("Process %d has exited with status 0", pid))

	executor("q")
	clear_variable()
}

func TestCallStack(t *testing.T) {
	var (
		execfile string
		err      error
		g        = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t4.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)
	pid := cmd.Process.Pid

	executor("b ./test_file/t4.go:6")

	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t4.go:6 breakpoint successfully"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring(`==>      6: 	return fmt.Sprintf("m = %d", m)`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("bt")
	g.Expect(outw.String()).Should(ContainSubstring(`test_file/t4.go:6`))
	g.Expect(outw.String()).Should(ContainSubstring(`test_file/t4.go:11 main.pppp1`))
	g.Expect(outw.String()).Should(ContainSubstring(`test_file/t4.go:16 main.main`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("n")
	g.Expect(outw.String()).Should(ContainSubstring(`==>     11: 	mstr := pppp2(m)`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("bt")
	g.Expect(outw.String()).Should(ContainSubstring(`test_file/t4.go:11`))
	g.Expect(outw.String()).Should(ContainSubstring(`test_file/t4.go:16 main.main`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("n")
	g.Expect(outw.String()).Should(ContainSubstring(`==>     12: 	fmt.Println(mstr)`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("bt")
	g.Expect(outw.String()).Should(ContainSubstring(`test_file/t4.go:12`))
	g.Expect(outw.String()).Should(ContainSubstring(`test_file/t4.go:16 main.main`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(Equal(""))
	g.Expect(errw.String()).Should(MatchRegexp("Process %d has exited with status 0", pid))

	executor("q")
	clear_variable()
}

func TestPrintString(t *testing.T) {
	var (
		execfile string
		err      error
		g        = NewGomegaWithT(t)
	)
	outw, errw := make_out_err()

	execfile, err = build_run_debug("./test_file/t5.go")
	g.Expect(err).Should(BeNil())
	defer os.Remove(execfile)
	pid := cmd.Process.Pid

	executor("b ./test_file/t5.go:11")

	g.Expect(outw.String()).Should(ContainSubstring("godbg add ./test_file/t5.go:11 breakpoint successfully"))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(ContainSubstring(`==>     11: 	godbgvint := uint64(100)`))
	g.Expect(errw.String()).Should(Equal(""))
	outw.Reset()

	executor("p godbgvstr")
	g.Expect(outw.String()).Should(ContainSubstring(`hello world`))
	outw.Reset()

	executor("c")
	g.Expect(outw.String()).Should(Equal(""))
	g.Expect(errw.String()).Should(MatchRegexp("Process %d has exited with status 0", pid))

	executor("q")
	clear_variable()
}