//go:build !debug

package utils

// WaitDebugger is a no-op on normal build
func WaitDebugger(string) {}
