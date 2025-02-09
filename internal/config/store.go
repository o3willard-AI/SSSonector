package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FileStore implements configuration storage using the filesystem
type FileStore struct {
	baseDir string
	logger  *zap.Logger
	mu      sync.RWMutex
}

// NewFileStore creates a new file-based configuration store
func NewFileStore(baseDir string, logger *zap.Logger) *FileStore {
	return &FileStore{
		baseDir: baseDir,
		logger:  logger,
	}
}

// Store stores a configuration
func (s *FileStore) Store(cfg *AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update metadata
	cfg.Metadata.UpdatedAt = time.Now()

	// Create directory for configuration type
	dir := filepath.Join(s.baseDir, "configs", string(cfg.Mode))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create filename with version
	filename := filepath.Join(dir, fmt.Sprintf("config-%s.json", cfg.Metadata.Version))

	// Marshal configuration
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write configuration file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Load loads a configuration by type and version
func (s *FileStore) Load(cfgType ConfigType, version string) (*AppConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find configuration file
	filename := filepath.Join(s.baseDir, "configs", string(cfgType), fmt.Sprintf("config-%s.json", version))

	// Read configuration file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse configuration
	cfg := &AppConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// Delete deletes a configuration by type and version
func (s *FileStore) Delete(cfgType ConfigType, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find configuration file
	filename := filepath.Join(s.baseDir, "configs", string(cfgType), fmt.Sprintf("config-%s.json", version))

	// Delete file
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete config file: %w", err)
	}

	return nil
}

// List lists all configurations
func (s *FileStore) List() ([]*AppConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var configs []*AppConfig

	// Walk configuration directory
	err := filepath.Walk(filepath.Join(s.baseDir, "configs"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip non-JSON files
		if !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		// Read configuration file
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		// Parse configuration
		cfg := &AppConfig{}
		if err := json.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}

		configs = append(configs, cfg)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	return configs, nil
}

// ListByType lists configurations by type
func (s *FileStore) ListByType(cfgType ConfigType) ([]*AppConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var configs []*AppConfig

	// Read configuration directory
	dir := filepath.Join(s.baseDir, "configs", string(cfgType))
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	// Read each configuration file
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Read configuration file
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// Parse configuration
		cfg := &AppConfig{}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

// ListVersions lists all versions of a configuration type
func (s *FileStore) ListVersions(cfgType ConfigType) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var versions []string

	// Read configuration directory
	dir := filepath.Join(s.baseDir, "configs", string(cfgType))
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	// Extract versions from filenames
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		version := strings.TrimSuffix(strings.TrimPrefix(file.Name(), "config-"), ".json")
		versions = append(versions, version)
	}

	// Sort versions
	sort.Strings(versions)

	return versions, nil
}

// GetLatest gets the latest version of a configuration type
func (s *FileStore) GetLatest(cfgType ConfigType) (*AppConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// List versions
	versions, err := s.ListVersions(cfgType)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no configurations found for type %s", cfgType)
	}

	// Get latest version
	latest := versions[len(versions)-1]

	// Load configuration
	return s.Load(cfgType, latest)
}
