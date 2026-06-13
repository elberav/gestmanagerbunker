//go:build linux

package backend

import (
	"os"
	"strings"
	"syscall"
)

// isBeingDebugged detecta si hay un debugger (ptrace) activo en Linux.
func isBeingDebugged() bool {
	status, err := os.ReadFile("/proc/self/status")
	if err == nil {
		if !strings.Contains(string(status), "TracerPid:\t0") {
			return true
		}
	}

	// Si nos pueden hacer ptrace, probablemente hay un debugger
	if err := syscall.PtraceTraceme(); err != nil {
		return true
	}

	return false
}

// HideFile no hace nada en Linux porque el punto al inicio del nombre ya lo oculta.
func HideFile(path string) {}
