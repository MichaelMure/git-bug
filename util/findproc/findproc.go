package findproc

import (
	"github.com/shirou/gopsutil/process"
	"os"
	"strings"
	"syscall"
)

// IsRunning tells if a git-bug process is running
func IsRunning(pid int32) bool {
	// never returns no error in a unix system
	findproc, err := os.FindProcess(int(pid))

	if err != nil {
		return false
	}

	// Signal 0 doesn't do anything but allow testing the process
	err = findproc.Signal(syscall.Signal(0))
	proc, _ := process.NewProcess(pid)
	procname, _ := proc.Name()
	if !strings.Contains(procname, "git-bug") {
		return false
	}

	// Todo: distinguish "you don't have access" and "process doesn't exist"

	return err == nil
}
