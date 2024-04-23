package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

const Version = "0.2.1"

func PrintVersionAndExit() {
	fmt.Println("Version:", Version)
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("Failed to read build info")
		os.Exit(1)
	}
	for _, m := range info.Settings {
		switch m.Key {
		case "vcs.revision":
			fmt.Printf("Git revision: %s\n", m.Value)
		case "vcs.time":
			fmt.Printf("Build time: %s\n", m.Value)
		}
	}
	os.Exit(0)
}
