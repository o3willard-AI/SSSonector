package main

import (
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// Config represents the application configuration
type Config struct {
	LogLevel string `yaml:"logLevel" json:"logLevel"`
	Server   struct {
		Host string `yaml:"host" json:"host"`
		Port int    `yaml:"port" json:"port"`
	} `yaml:"server" json:"server"`
	Connection struct {
		MaxConnections int            `yaml:"maxConnections" json:"maxConnections"`
		KeepAlive      bool           `yaml:"keepAlive" json:"keepAlive"`
		IdleTimeout    types.Duration `yaml:"idleTimeout" json:"idleTimeout"`
	} `yaml:"connection" json:"connection"`
	RateLimit struct {
		Enabled     bool  `yaml:"enabled" json:"enabled"`
		RequestRate int64 `yaml:"requestRate" json:"requestRate"`
		BurstSize   int   `yaml:"burstSize" json:"burstSize"`
	} `yaml:"rateLimit" json:"rateLimit"`
	CircuitBreaker struct {
		Enabled          bool           `yaml:"enabled" json:"enabled"`
		MaxFailures      int            `yaml:"maxFailures" json:"maxFailures"`
		ResetTimeout     types.Duration `yaml:"resetTimeout" json:"resetTimeout"`
		HalfOpenMaxCalls int            `yaml:"halfOpenMaxCalls" json:"halfOpenMaxCalls"`
	} `yaml:"circuitBreaker" json:"circuitBreaker"`
}

// Default configuration values
var defaultConfig = Config{
	LogLevel: "info",
	Server: struct {
		Host string `yaml:"host" json:"host"`
		Port int    `yaml:"port" json:"port"`
	}{
		Host: "localhost",
		Port: 8080,
	},
	Connection: struct {
		MaxConnections int            `yaml:"maxConnections" json:"maxConnections"`
		KeepAlive      bool           `yaml:"keepAlive" json:"keepAlive"`
		IdleTimeout    types.Duration `yaml:"idleTimeout" json:"idleTimeout"`
	}{
		MaxConnections: 1000,
		KeepAlive:      true,
		IdleTimeout:    types.NewDuration(5 * time.Minute),
	},
	RateLimit: struct {
		Enabled     bool  `yaml:"enabled" json:"enabled"`
		RequestRate int64 `yaml:"requestRate" json:"requestRate"`
		BurstSize   int   `yaml:"burstSize" json:"burstSize"`
	}{
		Enabled:     true,
		RequestRate: 1000,
		BurstSize:   100,
	},
	CircuitBreaker: struct {
		Enabled          bool           `yaml:"enabled" json:"enabled"`
		MaxFailures      int            `yaml:"maxFailures" json:"maxFailures"`
		ResetTimeout     types.Duration `yaml:"resetTimeout" json:"resetTimeout"`
		HalfOpenMaxCalls int            `yaml:"halfOpenMaxCalls" json:"halfOpenMaxCalls"`
	}{
		Enabled:          true,
		MaxFailures:      5,
		ResetTimeout:     types.NewDuration(30 * time.Second),
		HalfOpenMaxCalls: 2,
	},
}
