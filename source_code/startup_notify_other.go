//go:build !windows

package main

// notifyStartupFailure is a no-op outside Windows; the log file is still
// written by writeStartupError.
func notifyStartupFailure(title, message, logPath string) {}
