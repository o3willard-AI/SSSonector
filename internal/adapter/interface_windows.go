//go:build windows
// +build windows

package adapter

import (
	"fmt"
)

func New(name string, opts *Options) (Interface, error) {
	return nil, fmt.Errorf("windows platform not implemented")
}
