//go:build windows

package backend

import (
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")

	procIsDebuggerPresent            = kernel32.NewProc("IsDebuggerPresent")
	procCheckRemoteDebuggerPresent   = kernel32.NewProc("CheckRemoteDebuggerPresent")
	procNtQueryInformationProcess    = ntdll.NewProc("NtQueryInformationProcess")
	procNtSetInformationThread       = ntdll.NewProc("NtSetInformationThread")
	procGetThreadContext             = kernel32.NewProc("GetThreadContext")
	procGetCurrentThread             = kernel32.NewProc("GetCurrentThread")
	procCloseHandle                  = kernel32.NewProc("CloseHandle")
	procCreateToolhelp32Snapshot     = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First               = kernel32.NewProc("Process32FirstW")
	procProcess32Next                = kernel32.NewProc("Process32NextW")
)

const (
	ProcessDebugPort  = 7
	ProcessDebugFlags = 31

	ThreadHideFromDebugger = 17

	TH32CS_SNAPPROCESS = 0x00000002

	CONTEXT_DEBUG_REGISTERS = 0x00000008
)

type PROCESSENTRY32 struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [260]uint16
}

type CONTEXT struct {
	ContextFlags      uint32
	Dr0               uintptr
	Dr1               uintptr
	Dr2               uintptr
	Dr3               uintptr
	Dr6               uintptr
	Dr7               uintptr
	FloatSave         [80]byte
	SegGs             uint32
	SegFs             uint32
	SegEs             uint32
	SegDs             uint32
	Edi               uint32
	Esi               uint32
	Ebx               uint32
	Edx               uint32
	Ecx               uint32
	Eax               uint32
	Ebp               uint32
	Eip               uint32
	SegCs             uint32
	EFlags            uint32
	Esp               uint32
	SegSs             uint32
	ExtendedRegisters [512]byte
}

func isBeingDebugged() bool {
	if flag, _, _ := procIsDebuggerPresent.Call(); flag != 0 {
		return true
	}

	var remoteDebugger bool
	ret, _, _ := procCheckRemoteDebuggerPresent.Call(uintptr(0xffffffff), uintptr(unsafe.Pointer(&remoteDebugger)))
	if ret != 0 && remoteDebugger {
		return true
	}

	var debugPort uint32
	var retLen uint32
	status, _, _ := procNtQueryInformationProcess.Call(
		uintptr(0xffffffff),
		ProcessDebugPort,
		uintptr(unsafe.Pointer(&debugPort)),
		unsafe.Sizeof(debugPort),
		uintptr(unsafe.Pointer(&retLen)),
	)
	if status == 0 && debugPort != 0 {
		return true
	}

	var debugFlags uint32
	status, _, _ = procNtQueryInformationProcess.Call(
		uintptr(0xffffffff),
		ProcessDebugFlags,
		uintptr(unsafe.Pointer(&debugFlags)),
		unsafe.Sizeof(debugFlags),
		uintptr(unsafe.Pointer(&retLen)),
	)
	if status == 0 && debugFlags == 0 {
		return true
	}

	if hasDebuggerProcess() {
		return true
	}

	if hasHardwareBreakpoints() {
		return true
	}

	return false
}

func debuggerProcessNames() map[string]bool {
	return map[string]bool{
		"x64dbg.exe":     true,
		"x32dbg.exe":     true,
		"ollydbg.exe":    true,
		"windbg.exe":     true,
		"ida.exe":        true,
		"ida64.exe":      true,
		"immunitydbg.exe": true,
		"gdb.exe":        true,
		"devenv.exe":     true,
		"dnspy.exe":      true,
		"cheatengine.exe": true,
	}
}

func hasDebuggerProcess() bool {
	names := debuggerProcessNames()

	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == uintptr(0xffffffff) {
		return false
	}
	defer procCloseHandle.Call(snapshot)

	var pe PROCESSENTRY32
	pe.dwSize = uint32(unsafe.Sizeof(pe))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
	for ret != 0 {
		name := syscall.UTF16ToString(pe.szExeFile[:])
		if names[name] {
			return true
		}
		ret, _, _ = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
	}
	return false
}

func hasHardwareBreakpoints() bool {
	threadHandle, _, _ := procGetCurrentThread.Call()

	var ctx CONTEXT
	ctx.ContextFlags = CONTEXT_DEBUG_REGISTERS
	ret, _, _ := procGetThreadContext.Call(threadHandle, uintptr(unsafe.Pointer(&ctx)))
	if ret == 0 {
		return false
	}

	return ctx.Dr0 != 0 || ctx.Dr1 != 0 || ctx.Dr2 != 0 || ctx.Dr3 != 0
}

func hideFromDebugger() {
	procNtSetInformationThread.Call(
		uintptr(0xfffffffe),
		ThreadHideFromDebugger,
		0,
		0,
	)
}

func init() {
	hideFromDebugger()
}

// HideFile marca un archivo como oculto en Windows usando la API nativa.
func HideFile(path string) {
	ptr, _ := syscall.UTF16PtrFromString(path)
	syscall.SetFileAttributes(ptr, 0x02)
}


