//go:build windows

package backend

import (
	"syscall"
)

// isBeingDebugged detecta si hay un debugger activo en Windows usando la API nativa.
func isBeingDebugged() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	isDebuggerPresent := kernel32.NewProc("IsDebuggerPresent")
	
	// Llamada a IsDebuggerPresent() de Win32 API
	flag, _, _ := isDebuggerPresent.Call()
	return flag != 0
}

// HideFile marca un archivo como oculto en Windows usando la API nativa.
func HideFile(path string) {
	ptr, _ := syscall.UTF16PtrFromString(path)
	// 0x02 es el atributo FILE_ATTRIBUTE_HIDDEN
	syscall.SetFileAttributes(ptr, 0x02)
}
