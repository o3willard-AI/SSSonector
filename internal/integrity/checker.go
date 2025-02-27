package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	ConfigPermissions = 0644
	BinaryPermissions = 0755
)

// FileInfo stores information about a file
type FileInfo struct {
	Path       string
	Hash       string
	Size       int64
	ModTime    int64
	Mode       os.FileMode
	IsConfig   bool
	ConfigType string // "server" or "client"
}

// BinaryInfo stores information about a binary
type BinaryInfo = FileInfo

// GetFileInfo returns information about a file at the given path
func GetFileInfo(path string) (*FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open binary: %w", err)
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Determine if this is a config file
	isConfig := strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
	configType := ""
	if isConfig {
		// Read config to determine type
		var config struct {
			Type string `yaml:"type"`
		}
		if data, err := os.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(data, &config); err == nil {
				configType = config.Type
			}
		}
	}

	return &FileInfo{
		Path:       path,
		Hash:       hex.EncodeToString(hash.Sum(nil)),
		Size:       info.Size(),
		ModTime:    info.ModTime().Unix(),
		Mode:       info.Mode(),
		IsConfig:   isConfig,
		ConfigType: configType,
	}, nil
}

// VerifyFile verifies the integrity of a file against expected info
func VerifyFile(path string, expected *FileInfo) error {
	actual, err := GetFileInfo(path)
	if err != nil {
		return err
	}

	if actual.Hash != expected.Hash {
		return fmt.Errorf("hash mismatch for %s: expected %s, got %s",
			filepath.Base(path), expected.Hash, actual.Hash)
	}

	if actual.Size != expected.Size {
		return fmt.Errorf("size mismatch for %s: expected %d, got %d",
			filepath.Base(path), expected.Size, actual.Size)
	}

	// Check permissions
	expectedMode := expected.Mode
	if expected.IsConfig {
		expectedMode = ConfigPermissions
	} else {
		expectedMode = BinaryPermissions
	}

	if actual.Mode != expectedMode {
		return fmt.Errorf("permissions mismatch for %s: expected %o, got %o",
			filepath.Base(path), expectedMode, actual.Mode)
	}

	// For config files, verify type matches
	if expected.IsConfig && actual.ConfigType != expected.ConfigType {
		return fmt.Errorf("config type mismatch for %s: expected %s, got %s",
			filepath.Base(path), expected.ConfigType, actual.ConfigType)
	}

	return nil
}

// VerifyFiles verifies multiple files at once
func VerifyFiles(files map[string]*FileInfo) error {
	var firstErr error
	for path, expected := range files {
		if err := VerifyFile(path, expected); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// VerifyBinary is an alias for VerifyFile for backward compatibility
func VerifyBinary(path string, expected *BinaryInfo) error {
	return VerifyFile(path, expected)
}

// VerifyBinaries is an alias for VerifyFiles for backward compatibility
func VerifyBinaries(binaries map[string]*BinaryInfo) error {
	files := make(map[string]*FileInfo, len(binaries))
	for k, v := range binaries {
		files[k] = v
	}
	return VerifyFiles(files)
}

// FixConfigPermissions ensures config files have correct permissions
func FixConfigPermissions(configPath string) error {
	info, err := GetFileInfo(configPath)
	if err != nil {
		return fmt.Errorf("failed to get config info: %w", err)
	}

	if !info.IsConfig {
		return fmt.Errorf("not a config file: %s", configPath)
	}

	if info.Mode != ConfigPermissions {
		if err := os.Chmod(configPath, ConfigPermissions); err != nil {
			return fmt.Errorf("failed to set config permissions: %w", err)
		}
	}

	return nil
}
