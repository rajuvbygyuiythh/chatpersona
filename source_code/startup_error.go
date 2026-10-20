package main

// Startup failures in a -H windowsgui build are invisible: no console, no
// window. This helper always leaves a file behind and, on Windows, also shows a
// message box so the owner knows what happened.

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// writeStartupError appends one entry to startup-error.log next to the exe and
// returns the path it used. It never fails: if the exe folder is read-only the
// message still reaches the dialog.
func writeStartupError(dir, message string) string {
	path := filepath.Join(dir, "startup-error.log")
	entry := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return path
	}
	_, _ = f.WriteString(entry)
	_ = f.Close()
	return path
}
