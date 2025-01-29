package tunnel

import "errors"

var (
	// ErrCertManagerNotInitialized is returned when attempting to use TLS functions without initializing the certificate manager
	ErrCertManagerNotInitialized = errors.New("certificate manager not initialized")

	// ErrInvalidMode is returned when an invalid tunnel mode is specified
	ErrInvalidMode = errors.New("invalid tunnel mode (must be 'server' or 'client')")

	// ErrConnectionClosed is returned when attempting to use a closed connection
	ErrConnectionClosed = errors.New("connection closed")

	// ErrInterfaceNotInitialized is returned when attempting to use a tunnel without initializing the network interface
	ErrInterfaceNotInitialized = errors.New("network interface not initialized")

	// ErrInvalidConfiguration is returned when the tunnel configuration is invalid
	ErrInvalidConfiguration = errors.New("invalid tunnel configuration")
)
