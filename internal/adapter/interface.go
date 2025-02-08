package adapter

import (
	"io"
)

type Interface interface {
	io.ReadWriteCloser
	Configure(cfg *Config) error
	GetName() string
	GetMTU() int
	GetAddress() string
	IsUp() bool
	Cleanup() error
}

type Config struct {
	Name    string
	Address string
	MTU     int
}
