//go:build debug

package utils

import (
	"fmt"
	"os"
)

func WaitDebugger(reason string) {
	fmt.Printf("%s: debug %d\n", reason, os.Getpid())
	_, _ = fmt.Scanln()
}
