package adapter

import (
	"fmt"
)

func newWindowsInterface(name string) (Interface, error) {
	return nil, fmt.Errorf("windows platform not implemented")
}
