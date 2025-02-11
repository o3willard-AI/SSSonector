// Package store provides configuration storage implementations
package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// FileStore implements ConfigStore interface for file-based storage
type FileStore struct {
	configDir string
	watchers  []chan<- *types.AppConfig
	mu        sync.RWMutex
}

// NewFileStore creates a new FileStore instance
func NewFileStore(configDir string) *FileStore {
	return &FileStore{
		configDir: configDir,
		watchers:  make([]chan<- *types.AppConfig, 0),
	}
}

// Load loads configuration from file
func (s *FileStore) Load() (*types.AppConfig, error) {
	data, err := ioutil.ReadFile(filepath.Join(s.configDir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config types.AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}

// Store stores configuration to file
func (s *FileStore) Store(config *types.AppConfig) error {
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	filename := "config.yaml"
	path := filepath.Join(s.configDir, filename)

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// Notify watchers
	s.mu.RLock()
	for _, w := range s.watchers {
		select {
		case w <- config:
		default:
		}
	}
	s.mu.RUnlock()

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

	var versions []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".yaml") {
			data, err := ioutil.ReadFile(filepath.Join(s.configDir, f.Name()))
			if err != nil {
				continue
			}
			var cfg types.AppConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				continue
			}
			if cfg.Type == configType {
				versions = append(versions, cfg.Version)
			}
		}
	}

	sort.Strings(versions)
	return versions, nil
}

// Close implements io.Closer
func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, w := range s.watchers {
		close(w)
	}
	s.watchers = nil
	return nil
}

// Get returns the current configuration
func (s *FileStore) Get() (*types.AppConfig, error) {
	return s.Load()
}

// Set updates the current configuration
func (s *FileStore) Set(config *types.AppConfig) error {
	return s.Store(config)
}

// Update updates the configuration with the provided config
func (s *FileStore) Update(config *types.AppConfig) error {
	return s.Store(config)
}

// Watch returns a channel that receives configuration updates
func (s *FileStore) Watch() (<-chan *types.AppConfig, error) {
	ch := make(chan *types.AppConfig, 1)

	s.mu.Lock()
	s.watchers = append(s.watchers, ch)
	s.mu.Unlock()

	// Send initial configuration
	cfg, err := s.Load()
	if err != nil {
		return nil, err
	}

	select {
	case ch <- cfg:
	case <-time.After(time.Second):
		return nil, fmt.Errorf("timeout sending initial configuration")
	}

	return ch, nil
}
