//go:build linux || darwin
// +build linux darwin

package monitor

import (
	"syscall"
)

func killProcess(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGTERM)
		syscall.Kill(-pgid, syscall.SIGKILL)
	}
	return nil
}
