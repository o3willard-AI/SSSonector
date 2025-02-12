package pool

import (
	"context"
	"net"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// Factory defines a function type for creating new connections
type Factory func(ctx context.Context) (net.Conn, error)

// Config represents the configuration for a connection pool
type Config struct {
	// InitialSize is the initial number of connections to create
	InitialSize int `yaml:"initialSize" json:"initialSize"`

	// MaxSize is the maximum number of connections allowed in the pool
	MaxSize int `yaml:"maxSize" json:"maxSize"`

	// MinSize is the minimum number of connections to maintain
	MinSize int `yaml:"minSize" json:"minSize"`

	// MaxIdleTime is how long a connection can remain idle before being closed
	MaxIdleTime types.Duration `yaml:"maxIdleTime" json:"maxIdleTime"`

	// ConnectTimeout is the maximum time to wait for a new connection
	ConnectTimeout types.Duration `yaml:"connectTimeout" json:"connectTimeout"`

	// HealthCheck is a function to verify connection health
	HealthCheck func(net.Conn) error `yaml:"-" json:"-"`

	// Factory is used to create new connections
	Factory Factory `yaml:"-" json:"-"`

	// OnClose is called when a connection is closed
	OnClose func(net.Conn) `yaml:"-" json:"-"`
}
