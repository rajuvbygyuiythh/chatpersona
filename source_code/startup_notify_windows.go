//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32DLL      = syscall.NewLazyDLL("user32.dll")
	messageBoxProc = user32DLL.NewProc("MessageBoxW")
)

// notifyStartupFailure shows a native error box. A GUI build has no console, so
// this is the only way the owner can see why the window never appeared.
func notifyStartupFailure(title, message, logPath string) {
	text := message
	if logPath != "" {
		text += "\n\nDetails saved to:\n" + logPath
	}
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	const mbIconError = 0x00000010
	const mbSetForeground = 0x00010000
	_, _, _ = messageBoxProc.Call(0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(mbIconError|mbSetForeground))
}
