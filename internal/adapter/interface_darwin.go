//go:build darwin
// +build darwin

package adapter

import (
	"fmt"
)

func New(name string) (Interface, error) {
	return nil, fmt.Errorf("darwin platform not implemented")
}
