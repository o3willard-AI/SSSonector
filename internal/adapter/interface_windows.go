//go:build windows
// +build windows

package adapter

import (
	"fmt"
)

func New(name string) (Interface, error) {
	return nil, fmt.Errorf("windows platform not implemented")
}
