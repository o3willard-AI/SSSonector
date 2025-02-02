package adapter

import (
	"fmt"
)

func newDarwinInterface(name string) (Interface, error) {
	return nil, fmt.Errorf("darwin platform not implemented")
}
