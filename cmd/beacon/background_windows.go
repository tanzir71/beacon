package main

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

var getConsoleWindow = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow")

func backgroundConsoleWindow() uintptr {
	handle, _, _ := getConsoleWindow.Call()
	return handle
}

func backgroundExecutable(current string) string {
	if !strings.EqualFold(filepath.Base(current), "beacon-cli.exe") {
		return current
	}
	gui := filepath.Join(filepath.Dir(current), "beacon.exe")
	info, err := os.Stat(gui)
	if err == nil && !info.IsDir() {
		return gui
	}
	return current
}
