//go:build linux

package backend

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func isBeingDebugged() bool {
	status, err := os.ReadFile("/proc/self/status")
	if err == nil {
		if !strings.Contains(string(status), "TracerPid:\t0") {
			return true
		}

		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "PPid:") {
				parts := strings.Fields(line)
				if len(parts) == 2 {
					ppid, _ := strconv.Atoi(parts[1])
					if ppid > 1 && isProcessDebugger(ppid) {
						return true
					}
				}
			}
		}
	}

	if err := syscall.PtraceTraceme(); err != nil {
		return true
	}

	if hasDebuggerProcess() {
		return true
	}

	if hasSuspiciousEnv() {
		return true
	}

	return false
}

func debuggerProcessNames() map[string]bool {
	return map[string]bool{
		"gdb":            true,
		"lldb":           true,
		"lldb-server":    true,
		"strace":         true,
		"ltrace":         true,
		"valgrind":       true,
		"rr":             true,
		"perf":           true,
		"gdbserver":      true,
		"dlv":            true,
		"rade":           true,
		"frida":          true,
		"frida-server":   true,
	}
}

func isProcessDebugger(pid int) bool {
	comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return false
	}
	name := strings.TrimSpace(string(comm))
	return debuggerProcessNames()[name]
}

func hasDebuggerProcess() bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		if pid == os.Getpid() || pid == os.Getppid() {
			continue
		}
		if isProcessDebugger(pid) {
			return true
		}
	}
	return false
}

func hasSuspiciousEnv() bool {
	for _, env := range []string{"LD_PRELOAD", "LD_LIBRARY_PATH"} {
		if val := os.Getenv(env); val != "" {
			lower := strings.ToLower(val)
			if strings.Contains(lower, "inject") ||
				strings.Contains(lower, "hook") ||
				strings.Contains(lower, "trace") ||
				strings.Contains(lower, "debug") {
				return true
			}
		}
	}
	return false
}

func earlyAntiDebug() {
	syscall.PtraceTraceme()
}

func HideFile(path string) {}
