//go:build windows

package prompt

import (
	"syscall"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleTitle = kernel32.NewProc("GetConsoleTitleW")
	procSetConsoleTitle = kernel32.NewProc("SetConsoleTitleW")
)

// savedTitle is the console title captured at the first SetTitle, so
// RestoreTitle can hand the window back untouched.
var savedTitle string

func setTerminalTitle(title string) {
	if savedTitle == "" {
		buf := make([]uint16, 512)
		n, _, _ := procGetConsoleTitle.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n > 0 {
			savedTitle = syscall.UTF16ToString(buf[:n])
		}
	}
	setConsoleTitle(title)
}

func restoreTerminalTitle() {
	if savedTitle != "" {
		setConsoleTitle(savedTitle)
		savedTitle = ""
	}
}

func setConsoleTitle(title string) {
	p, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	procSetConsoleTitle.Call(uintptr(unsafe.Pointer(p)))
}
