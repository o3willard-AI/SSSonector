// Package store provides configuration storage implementations
package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// FileStore implements ConfigStore interface for file-based storage
type FileStore struct {
	configDir string
}

// NewFileStore creates a new FileStore instance
func NewFileStore(configDir string) *FileStore {
	return &FileStore{configDir: configDir}
}

// Load loads configuration from file
func (s *FileStore) Load() (*types.AppConfig, error) {
	files, err := ioutil.ReadDir(s.configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return types.DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}

	var latest string
	var latestTime int64
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			if f.ModTime().Unix() > latestTime {
				latest = f.Name()
				latestTime = f.ModTime().Unix()
			}
		}
	}

	if latest == "" {
		return types.DefaultConfig(), nil
	}

	data, err := ioutil.ReadFile(filepath.Join(s.configDir, latest))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config types.AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}

// Store stores configuration to file
func (s *FileStore) Store(config *types.AppConfig) error {
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	filename := fmt.Sprintf("config-%s-%s.json", config.Type, config.Version)
	path := filepath.Join(s.configDir, filename)

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// ListVersions lists all available configuration versions for a given type
func (s *FileStore) ListVersions(configType types.Type) ([]string, error) {
	files, err := ioutil.ReadDir(s.configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}

	prefix := fmt.Sprintf("config-%s-", configType)
	var versions []string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ".json") {
			version := strings.TrimPrefix(strings.TrimSuffix(f.Name(), ".json"), prefix)
			versions = append(versions, version)
		}
	}

	sort.Strings(versions)
	return versions, nil
}
