package bp

import (
	"fmt"
	sys "golang.org/x/sys/unix"
	"reflect"
	//	"syscall"
)

type Info struct {
	Pid        int
	Original   []byte
	Filename   string
	Filelineno string
	Addr       uintptr
}

type BP struct {
	bpaddr []*Info
}

func (p *BP) SetBreakpoint(bpInfo *Info) error {
	original := make([]byte, 1)
	curBpInfo := bpInfo
	_, err := sys.PtracePeekData(curBpInfo.Pid, curBpInfo.Addr, original)
	if err != nil {
		fmt.Printf("PtracePeekData %s\n", err.Error())
		return err
	}
	_, err = sys.PtracePokeData(curBpInfo.Pid, curBpInfo.Addr, []byte{0xCC})
	if err != nil {
		fmt.Printf("PtracePokeData %s\n", err.Error())
		return err
	}
	curBpInfo.Original = original
	p.bpaddr = append(p.bpaddr, curBpInfo)
	return nil
}

func (p *BP) ListBreakpoint() {
	if nil == p.bpaddr {
		fmt.Printf("No breakpoint list\n")
		return
	}
	fmt.Printf("BreakPoint list:\n")
	for k, v := range p.bpaddr {
		fmt.Printf("\t %d. %s:%s\n", k, v.Filename, v.Filelineno)
	}
}

func (p *BP) GetPC(pid int) (uint64, error) {
	var (
		regs *sys.PtraceRegs
		err  error
	)
	if regs, err = p.getRegs(pid); err != nil {
		return 0, err
	}
	return regs.PC(), nil
}

func (p *BP) ListRegs(pid int) error {
	var (
		regs *sys.PtraceRegs
		err  error
	)
	if regs, err = p.getRegs(pid); err != nil {
		return err
	}
	v := reflect.ValueOf(regs).Elem()
	k := v.Type()
	fmt.Printf("List Regs:\n")
	for i := 0; i < v.NumField(); i++ {
		key := k.Field(i)
		val := v.Field(i)
		fmt.Printf("\t%-8s = %-3d\n", key.Name, val.Interface())
	}
	fmt.Printf("\t%-8s = %-3d\n", "PC", regs.PC())
	return nil
}

func (p *BP) getRegs(pid int) (*sys.PtraceRegs, error) {
	regs := &sys.PtraceRegs{}
	if err := sys.PtraceGetRegs(pid, regs); err != nil {
		return nil, err
	}
	return regs, nil
}

func ClearBreakPoint(pid int, addr uintptr) error {
	return nil
}

func Continue(pid int) error {
	if err := sys.PtraceCont(pid, 0); err != nil {
		return err
	}
	return nil
}
