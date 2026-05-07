//go:build linux

package backend

import (
	"os"
	"strings"
)

// isBeingDebugged detecta si hay un debugger (ptrace) activo en Linux.
func isBeingDebugged() bool {
	status, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return false
	}
	// Si TracerPid es distinto de 0, hay un debugger enganchado.
	return !strings.Contains(string(status), "TracerPid:\t0")
}

// HideFile no hace nada en Linux porque el punto al inicio del nombre ya lo oculta.
func HideFile(path string) {}
