package states

import (
	"context"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestInitializingHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("missing auth config", func(t *testing.T) {
		cfg := &types.AppConfig{
			Type:    types.TypeServer.String(),
			Version: "1.0.0",
			Config: types.ServiceConfig{
				Mode: string(types.ModeServer),
			},
		}
		handler := NewInitializingHandler(logger, cfg)
		err := handler.checkFilePermissions(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "auth configuration is missing")
	})

	t.Run("missing files", func(t *testing.T) {
		cfg := &types.AppConfig{
			Type:    types.TypeServer.String(),
			Version: "1.0.0",
			Config: types.ServiceConfig{
				Mode: string(types.ModeServer),
				Auth: types.AuthConfig{
					CertFile: "testdata/cert.pem",
					KeyFile:  "testdata/key.pem",
					CAFile:   "testdata/ca.pem",
				},
			},
		}
		handler := NewInitializingHandler(logger, cfg)
		err := handler.checkFilePermissions(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file access error")
	})

	t.Run("system resources", func(t *testing.T) {
		cfg := &types.AppConfig{
			Type:    types.TypeServer.String(),
			Version: "1.0.0",
			Config: types.ServiceConfig{
				Mode: string(types.ModeServer),
			},
		}
		handler := NewInitializingHandler(logger, cfg)
		err := handler.checkSystemResources(context.Background())
		assert.NoError(t, err)
	})

	t.Run("network availability", func(t *testing.T) {
		cfg := &types.AppConfig{
			Type:    types.TypeServer.String(),
			Version: "1.0.0",
			Config: types.ServiceConfig{
				Mode: string(types.ModeServer),
			},
		}
		handler := NewInitializingHandler(logger, cfg)
		err := handler.checkNetworkAvailability(context.Background())
		assert.NoError(t, err)
	})
}

func TestRunningHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	cfg := &types.AppConfig{
		Type:    types.TypeServer.String(),
		Version: "1.0.0",
		Config: types.ServiceConfig{
			Mode: string(types.ModeClient),
			Network: types.NetworkConfig{
				Interface: "lo",
				Address:   "127.0.0.1/8",
			},
			Tunnel: types.TunnelConfig{
				ServerAddress: "localhost",
				ServerPort:    8080,
			},
		},
	}

	t.Run("process health", func(t *testing.T) {
		handler := NewRunningHandler(logger, cfg)
		err := handler.checkProcessHealth(context.Background())
		assert.NoError(t, err)
	})

	t.Run("resource usage", func(t *testing.T) {
		handler := NewRunningHandler(logger, cfg)
		err := handler.checkResourceUsage(context.Background())
		assert.NoError(t, err)
	})

	t.Run("connectivity", func(t *testing.T) {
		handler := NewRunningHandler(logger, cfg)
		err := handler.checkConnectivity(context.Background())
		assert.NoError(t, err)
	})

	t.Run("state lifecycle", func(t *testing.T) {
		handler := NewRunningHandler(logger, cfg)

		// Test OnEnter
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := handler.OnEnter(ctx)
		assert.NoError(t, err)

		// Verify monitoring is active
		assert.NotNil(t, handler.connectivityMonitor)
		assert.NotNil(t, handler.monitorCancel)

		// Test OnExit
		err = handler.OnExit(ctx)
		assert.NoError(t, err)

		// Verify monitoring is cleaned up
		assert.Nil(t, handler.monitorCancel)

		// Verify context is not cancelled
		select {
		case <-ctx.Done():
			t.Error("context should not be cancelled")
		default:
			// Success
		}
	})

	t.Run("connectivity monitoring", func(t *testing.T) {
		handler := NewRunningHandler(logger, cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := handler.OnEnter(ctx)
		assert.NoError(t, err)

		// Wait for a monitoring cycle
		time.Sleep(time.Second)

		// Verify no errors occurred
		select {
		case err := <-handler.connectivityMonitor:
			t.Errorf("unexpected error from monitor: %v", err)
		default:
			// Success - no errors
		}

		err = handler.OnExit(ctx)
		assert.NoError(t, err)
	})
}

func TestStoppingHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &types.AppConfig{
		Type:    types.TypeServer.String(),
		Version: "1.0.0",
		Config: types.ServiceConfig{
			Mode: string(types.ModeServer),
		},
	}

	t.Run("cleanup tasks", func(t *testing.T) {
		handler := NewStoppingHandler(logger, cfg)
		err := handler.OnEnter(context.Background())
		assert.NoError(t, err)
	})

	t.Run("validate cleanup", func(t *testing.T) {
		handler := NewStoppingHandler(logger, cfg)
		err := handler.Validate(context.Background())
		assert.NoError(t, err)
	})
}
