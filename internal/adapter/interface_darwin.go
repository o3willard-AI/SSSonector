//go:build darwin
// +build darwin

package adapter

import (
	"fmt"
)

func New(name string, opts *Options) (Interface, error) {
	return nil, fmt.Errorf("darwin platform not implemented")
}
