package tunnel

import "crypto/tls"

// Modern cipher suites with perfect forward secrecy
var modernCipherSuites = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
}

// Modern TLS configuration based on Mozilla's recommendations
var modernTLSConfig = &tls.Config{
	MinVersion:               tls.VersionTLS12,
	PreferServerCipherSuites: true,
	CipherSuites:             modernCipherSuites,
	CurvePreferences: []tls.CurveID{
		tls.X25519,
		tls.CurveP384,
		tls.CurveP256,
	},
	SessionTicketsDisabled: true, // Disable session tickets for perfect forward secrecy
	Renegotiation:          tls.RenegotiateNever,
}

// GetModernTLSConfig returns a secure TLS configuration
func GetModernTLSConfig(baseConfig *tls.Config) *tls.Config {
	// Start with modern defaults
	config := modernTLSConfig.Clone()

	// Copy over certificate settings from base config
	if baseConfig != nil {
		config.GetCertificate = baseConfig.GetCertificate
		config.GetClientCertificate = baseConfig.GetClientCertificate
		config.ClientAuth = baseConfig.ClientAuth
		config.ClientCAs = baseConfig.ClientCAs
		config.RootCAs = baseConfig.RootCAs
		config.VerifyConnection = baseConfig.VerifyConnection
	}

	return config
}

// SecurityLevel represents TLS security profile
type SecurityLevel int

const (
	// SecurityModern uses modern cipher suites and settings
	SecurityModern SecurityLevel = iota
	// SecurityIntermediate supports older clients while maintaining security
	SecurityIntermediate
	// SecurityOld supports very old clients (not recommended)
	SecurityOld
)

// GetTLSSecurityLevel returns appropriate TLS config for security level
func GetTLSSecurityLevel(level SecurityLevel, baseConfig *tls.Config) *tls.Config {
	switch level {
	case SecurityModern:
		return GetModernTLSConfig(baseConfig)
	case SecurityIntermediate:
		config := GetModernTLSConfig(baseConfig)
		// Add some older but still secure cipher suites
		config.CipherSuites = append(config.CipherSuites,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		)
		return config
	case SecurityOld:
		config := GetModernTLSConfig(baseConfig)
		config.MinVersion = tls.VersionTLS10
		// Add older cipher suites (not recommended)
		config.CipherSuites = append(config.CipherSuites,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		)
		return config
	default:
		return GetModernTLSConfig(baseConfig)
	}
}
