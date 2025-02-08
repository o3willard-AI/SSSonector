//go:build windows
// +build windows

package monitor

import (
	"os"
)

func killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err == nil {
		process.Kill()
	}
	return nil
}
