package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/config/validator"
)

func main() {
	fmt.Println("🚀 SSSonector Configuration Validation Test Suite")
	fmt.Println("==============================================\n")

	// Test 1: Valid configuration
	fmt.Println("📋 Test 1: Valid Configuration")
	testValidConfig()

	// Test 2: Missing schema version
	fmt.Println("\n📋 Test 2: Missing Schema Version")
	testMissingSchemaVersion()

	// Test 3: Invalid migration history
	fmt.Println("\n📋 Test 3: Invalid Migration History")
	testInvalidMigrationHistory()

	// Test 4: Invalid certificate paths
	fmt.Println("\n📋 Test 4: Invalid Certificate Paths")
	testInvalidCertificatePaths()

	// Test 5: Invalid TLS configuration
	fmt.Println("\n📋 Test 5: Invalid TLS Configuration")
	testInvalidTLSConfig()

	// Test 6: Invalid throttle configuration
	fmt.Println("\n📋 Test 6: Invalid Throttle Configuration")
	testInvalidThrottleConfig()

	// Test 7: Invalid environment configuration
	fmt.Println("\n📋 Test 7: Invalid Environment Configuration")
	testInvalidEnvironmentConfig()

	// Test 8: IP address validation
	fmt.Println("\n📋 Test 8: IP Address Validation")
	testIPAddressValidation()

	// Test 9: Invalid certificate extensions
	fmt.Println("\n📋 Test 9: Invalid Certificate Extensions")
	testInvalidCertificateExtensions()

	fmt.Println("\n🎉 All tests completed!")
}

func testValidConfig() {
	configData, err := os.ReadFile("test_config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config types.AppConfig
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	validator := validator.NewValidator()
	err = validator.Validate(&config)
	if err != nil {
		fmt.Printf("❌ Valid config failed validation: %v\n", err)
	} else {
		fmt.Printf("✅ Valid config passed validation\n")
		fmt.Printf("   Schema Version: %s\n", config.Metadata.SchemaVersion)
		fmt.Printf("   Migration Records: %d\n", len(config.Metadata.MigrationHistory))
	}
}

func testMissingSchemaVersion() {
	validator := validator.NewValidator()

	invalidConfig := types.AppConfig{
		Version: "1.0.0",
		Metadata: types.ConfigMetadata{
			Version:     "1.0.0",
			Created:     time.Now(),
			Modified:    time.Now(),
			CreatedBy:   "test",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environment: "development",
			Region:      "local",
			// SchemaVersion intentionally missing
			MigrationHistory: []types.MigrationRecord{},
		},
		Config: &types.Config{
			Mode: "server",
		},
		Throttle: types.ThrottleConfig{
			Enabled: false,
			Rate:    1024,
			Burst:   1024,
		},
	}

	err := validator.Validate(&invalidConfig)
	if err != nil {
		fmt.Printf("✅ Missing schema version correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Missing schema version was accepted (should fail)")
	}
}

func testInvalidMigrationHistory() {
	validator := validator.NewValidator()

	invalidConfig := types.AppConfig{
		Version: "1.0.0",
		Metadata: types.ConfigMetadata{
			Version:       "1.0.0",
			Created:       time.Now(),
			Modified:      time.Now(),
			CreatedBy:     "test",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Environment:   "development",
			Region:        "local",
			SchemaVersion: "1.0.0",
			MigrationHistory: []types.MigrationRecord{
				{
					FromVersion: "0.0.0",
					// ToVersion intentionally missing
					Timestamp: time.Now(),
					Status:    "completed",
					Notes:     "Test migration",
				},
			},
		},
		Config: &types.Config{
			Mode: "server",
		},
		Throttle: types.ThrottleConfig{
			Enabled: false,
			Rate:    1024,
			Burst:   1024,
		},
	}

	err := validator.Validate(&invalidConfig)
	if err != nil {
		fmt.Printf("✅ Invalid migration history correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid migration history was accepted (should fail)")
	}
}

func testInvalidCertificatePaths() {
	validator := validator.NewValidator()

	invalidConfig := types.AppConfig{
		Version: "1.0.0",
		Metadata: types.ConfigMetadata{
			Version:       "1.0.0",
			Created:       time.Now(),
			Modified:      time.Now(),
			CreatedBy:     "test",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Environment:   "development",
			Region:        "local",
			SchemaVersion: "1.0.0",
			MigrationHistory: []types.MigrationRecord{
				{
					FromVersion: "0.0.0",
					ToVersion:   "1.0.0",
					Timestamp:   time.Now(),
					Status:      "completed",
					Notes:       "Test migration",
				},
			},
		},
		Config: &types.Config{
			Mode: "server",
			Auth: types.AuthConfig{
				CertFile: "../etc/sssonector/certs/server.crt", // Invalid path with ..
				KeyFile:  "/etc/sssonector/certs/server.key",
				CAFile:   "/etc/sssonector/certs/ca.crt",
			},
		},
		Throttle: types.ThrottleConfig{
			Enabled: false,
			Rate:    1024,
			Burst:   1024,
		},
	}

	err := validator.Validate(&invalidConfig)
	if err != nil {
		fmt.Printf("✅ Invalid certificate path correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid certificate path was accepted (should fail)")
	}
}

func testInvalidTLSConfig() {
	validator := validator.NewValidator()

	invalidConfig := types.AppConfig{
		Version: "1.0.0",
		Metadata: types.ConfigMetadata{
			Version:       "1.0.0",
			Created:       time.Now(),
			Modified:      time.Now(),
			CreatedBy:     "test",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Environment:   "development",
			Region:        "local",
			SchemaVersion: "1.0.0",
			MigrationHistory: []types.MigrationRecord{
				{
					FromVersion: "0.0.0",
					ToVersion:   "1.0.0",
					Timestamp:   time.Now(),
					Status:      "completed",
					Notes:       "Test migration",
				},
			},
		},
		Config: &types.Config{
			Mode: "server",
			Security: types.SecurityConfig{
				TLS: types.TLSConfigOptions{
					MinVersion: "1.3",
					MaxVersion: "1.2", // Min > Max - invalid
				},
			},
		},
		Throttle: types.ThrottleConfig{
			Enabled: false,
			Rate:    1024,
			Burst:   1024,
		},
	}

	err := validator.Validate(&invalidConfig)
	if err != nil {
		fmt.Printf("✅ Invalid TLS config correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid TLS config was accepted (should fail)")
	}
}

func testInvalidThrottleConfig() {
	validator := validator.NewValidator()

	invalidConfig := types.AppConfig{
		Version: "1.0.0",
		Metadata: types.ConfigMetadata{
			Version:       "1.0.0",
			Created:       time.Now(),
			Modified:      time.Now(),
			CreatedBy:     "test",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Environment:   "development",
			Region:        "local",
			SchemaVersion: "1.0.0",
			MigrationHistory: []types.MigrationRecord{
				{
					FromVersion: "0.0.0",
					ToVersion:   "1.0.0",
					Timestamp:   time.Now(),
					Status:      "completed",
					Notes:       "Test migration",
				},
			},
		},
		Config: &types.Config{
			Mode: "server",
		},
		Throttle: types.ThrottleConfig{
			Enabled: true,
			Rate:    1024,
			Burst:   1024 * 100, // 100x rate - invalid
		},
	}

	err := validator.Validate(&invalidConfig)
	if err != nil {
		fmt.Printf("✅ Invalid throttle config correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid throttle config was accepted (should fail)")
	}
}

func testInvalidEnvironmentConfig() {
	validator := validator.NewValidator()

	invalidConfig := types.AppConfig{
		Version: "1.0.0",
		Metadata: types.ConfigMetadata{
			Version:       "1.0.0",
			Created:       time.Now(),
			Modified:      time.Now(),
			CreatedBy:     "test",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Environment:   "invalid-env", // Invalid environment
			Region:        "local",
			SchemaVersion: "1.0.0",
			MigrationHistory: []types.MigrationRecord{
				{
					FromVersion: "0.0.0",
					ToVersion:   "1.0.0",
					Timestamp:   time.Now(),
					Status:      "completed",
					Notes:       "Test migration",
				},
			},
		},
		Config: &types.Config{
			Mode: "server",
		},
		Throttle: types.ThrottleConfig{
			Enabled: false,
			Rate:    1024,
			Burst:   1024,
		},
	}

	err := validator.Validate(&invalidConfig)
	if err != nil {
		fmt.Printf("✅ Invalid environment config correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid environment config was accepted (should fail)")
	}
}

func testIPAddressValidation() {
	validator := validator.NewValidator()

	// Test valid IP
	err := validator.ValidateIPAddress("192.168.1.1")
	if err != nil {
		fmt.Printf("❌ Valid IP rejected: %v\n", err)
	} else {
		fmt.Println("✅ Valid IP address accepted")
	}

	// Test invalid IP
	err = validator.ValidateIPAddress("999.999.999.999")
	if err != nil {
		fmt.Printf("✅ Invalid IP correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid IP was accepted (should fail)")
	}

	// Test CIDR
	err = validator.ValidateCIDR("192.168.1.0/24")
	if err != nil {
		fmt.Printf("❌ Valid CIDR rejected: %v\n", err)
	} else {
		fmt.Println("✅ Valid CIDR notation accepted")
	}

	// Test invalid CIDR
	err = validator.ValidateCIDR("999.999.999.999/24")
	if err != nil {
		fmt.Printf("✅ Invalid CIDR correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid CIDR was accepted (should fail)")
	}
}

func testInvalidCertificateExtensions() {
	validator := validator.NewValidator()

	invalidConfig := types.AppConfig{
		Version: "1.0.0",
		Metadata: types.ConfigMetadata{
			Version:       "1.0.0",
			Created:       time.Now(),
			Modified:      time.Now(),
			CreatedBy:     "test",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Environment:   "development",
			Region:        "local",
			SchemaVersion: "1.0.0",
			MigrationHistory: []types.MigrationRecord{
				{
					FromVersion: "0.0.0",
					ToVersion:   "1.0.0",
					Timestamp:   time.Now(),
					Status:      "completed",
					Notes:       "Test migration",
				},
			},
		},
		Config: &types.Config{
			Mode: "server",
			Auth: types.AuthConfig{
				CertFile: "/etc/sssonector/certs/server.txt", // Invalid extension
				KeyFile:  "/etc/sssonector/certs/server.key",
				CAFile:   "/etc/sssonector/certs/ca.crt",
			},
		},
		Throttle: types.ThrottleConfig{
			Enabled: false,
			Rate:    1024,
			Burst:   1024,
		},
	}

	err := validator.Validate(&invalidConfig)
	if err != nil {
		fmt.Printf("✅ Invalid certificate extension correctly rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid certificate extension was accepted (should fail)")
	}
}
