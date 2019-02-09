package bp

import (
	"fmt"
	sys "golang.org/x/sys/unix"
	//	"syscall"
)

func SetBreakpoint(pid int, addr uintptr) ([]byte, error) {
	original := make([]byte, 1)
	_, err := sys.PtracePeekData(pid, addr, original)
	if err != nil {
		fmt.Printf("PtracePeekData %s\n", err.Error())
		return nil, err
	}
	_, err = sys.PtracePokeData(pid, addr, []byte{0xCC})
	if err != nil {
		fmt.Printf("PtracePokeData %s\n", err.Error())
		return nil, err
	}
	return original, nil
}

func Continue(pid int) error {
	if err := sys.PtraceCont(pid, 0); err != nil {
		return err
	}
	return nil
}
