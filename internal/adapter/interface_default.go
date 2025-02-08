//go:build !linux && !darwin && !windows
// +build !linux,!darwin,!windows

package adapter

import (
	"fmt"
	"runtime"
)

func New(name string) (Interface, error) {
	return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}
