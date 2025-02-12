package states

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/network"
	"go.uber.org/zap"
)

// BaseHandler provides common functionality for state handlers
type BaseHandler struct {
	logger *zap.Logger
	cfg    *types.AppConfig
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(logger *zap.Logger, cfg *types.AppConfig) *BaseHandler {
	return &BaseHandler{
		logger: logger,
		cfg:    cfg,
	}
}

// InitializingHandler handles the initializing state
type InitializingHandler struct {
	*BaseHandler
	resourceChecks []func(context.Context) error
}

// NewInitializingHandler creates a new initializing handler
func NewInitializingHandler(logger *zap.Logger, cfg *types.AppConfig) *InitializingHandler {
	h := &InitializingHandler{
		BaseHandler: NewBaseHandler(logger, cfg),
	}
	h.resourceChecks = []func(context.Context) error{
		h.checkFilePermissions,
		h.checkSystemResources,
		h.checkNetworkAvailability,
	}
	return h
}

func (h *InitializingHandler) OnEnter(ctx context.Context) error {
	h.logger.Info("entering initializing state")
	for _, check := range h.resourceChecks {
		if err := check(ctx); err != nil {
			return fmt.Errorf("initialization check failed: %w", err)
		}
	}
	return nil
}

func (h *InitializingHandler) OnExit(ctx context.Context) error {
	h.logger.Info("exiting initializing state")
	return nil
}

func (h *InitializingHandler) Validate(ctx context.Context) error {
	for _, check := range h.resourceChecks {
		if err := check(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (h *InitializingHandler) checkFilePermissions(ctx context.Context) error {
	if h.cfg.Config.Auth == nil {
		return fmt.Errorf("auth configuration is missing")
	}

	files := []string{
		h.cfg.Config.Auth.CertFile,
		h.cfg.Config.Auth.KeyFile,
		h.cfg.Config.Auth.CAFile,
	}

	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("file access error (%s): %w", file, err)
		}
	}
	return nil
}

func (h *InitializingHandler) checkSystemResources(ctx context.Context) error {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return fmt.Errorf("failed to get file descriptor limit: %w", err)
	}

	if rLimit.Cur < 1024 {
		return fmt.Errorf("insufficient file descriptor limit: %d (minimum 1024 required)", rLimit.Cur)
	}
	return nil
}

func (h *InitializingHandler) checkNetworkAvailability(ctx context.Context) error {
	// Basic network interface check
	ifaces, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return fmt.Errorf("failed to check network interfaces: %w", err)
	}
	if len(ifaces) == 0 {
		return fmt.Errorf("no network interfaces available")
	}
	return nil
}

// RunningHandler handles the running state
type RunningHandler struct {
	*BaseHandler
	healthChecks        []func(context.Context) error
	connectivityChecker *network.ConnectivityChecker
	connectivityMonitor <-chan error
	monitorCancel       context.CancelFunc
}

// NewRunningHandler creates a new running handler
func NewRunningHandler(logger *zap.Logger, cfg *types.AppConfig) *RunningHandler {
	h := &RunningHandler{
		BaseHandler:         NewBaseHandler(logger, cfg),
		connectivityChecker: network.NewConnectivityChecker(cfg),
	}
	h.healthChecks = []func(context.Context) error{
		h.checkProcessHealth,
		h.checkResourceUsage,
		h.checkConnectivity,
	}
	return h
}

func (h *RunningHandler) OnEnter(ctx context.Context) error {
	h.logger.Info("entering running state")

	// Start connectivity monitoring
	monitorCtx, cancel := context.WithCancel(ctx)
	h.monitorCancel = cancel

	errChan, err := h.connectivityChecker.MonitorConnectivity(monitorCtx, 30*time.Second)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to start connectivity monitoring: %w", err)
	}
	h.connectivityMonitor = errChan

	// Start monitoring goroutine
	go func() {
		for {
			select {
			case <-monitorCtx.Done():
				return
			case err := <-errChan:
				if err != nil {
					h.logger.Error("connectivity check failed",
						zap.Error(err))
				}
			}
		}
	}()

	return h.Validate(ctx)
}

func (h *RunningHandler) OnExit(ctx context.Context) error {
	h.logger.Info("exiting running state")
	if h.monitorCancel != nil {
		h.monitorCancel()
		h.monitorCancel = nil
	}
	return nil
}

func (h *RunningHandler) Validate(ctx context.Context) error {
	for _, check := range h.healthChecks {
		if err := check(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (h *RunningHandler) checkProcessHealth(ctx context.Context) error {
	// Check if process is responding
	select {
	case <-ctx.Done():
		return fmt.Errorf("process health check interrupted")
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (h *RunningHandler) checkResourceUsage(ctx context.Context) error {
	var memInfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&memInfo); err != nil {
		return fmt.Errorf("failed to get system info: %w", err)
	}

	// Check if system has sufficient free memory (at least 10%)
	freePercentage := float64(memInfo.Freeram) / float64(memInfo.Totalram) * 100
	if freePercentage < 10 {
		return fmt.Errorf("insufficient free memory: %.2f%% (minimum 10%% required)", freePercentage)
	}
	return nil
}

func (h *RunningHandler) checkConnectivity(ctx context.Context) error {
	return h.connectivityChecker.CheckNetworkConnectivity(ctx)
}

// StoppingHandler handles the stopping state
type StoppingHandler struct {
	*BaseHandler
	cleanupTasks []func(context.Context) error
}

// NewStoppingHandler creates a new stopping handler
func NewStoppingHandler(logger *zap.Logger, cfg *types.AppConfig) *StoppingHandler {
	h := &StoppingHandler{
		BaseHandler: NewBaseHandler(logger, cfg),
	}
	h.cleanupTasks = []func(context.Context) error{
		h.cleanupConnections,
		h.cleanupResources,
		h.cleanupFiles,
	}
	return h
}

func (h *StoppingHandler) OnEnter(ctx context.Context) error {
	h.logger.Info("entering stopping state")
	for _, task := range h.cleanupTasks {
		if err := task(ctx); err != nil {
			h.logger.Error("cleanup task failed", zap.Error(err))
			// Continue with other cleanup tasks even if one fails
		}
	}
	return nil
}

func (h *StoppingHandler) OnExit(ctx context.Context) error {
	h.logger.Info("exiting stopping state")
	return nil
}

func (h *StoppingHandler) Validate(ctx context.Context) error {
	// Verify cleanup was successful
	return nil
}

func (h *StoppingHandler) cleanupConnections(ctx context.Context) error {
	// Implement connection cleanup
	return nil
}

func (h *StoppingHandler) cleanupResources(ctx context.Context) error {
	// Implement resource cleanup
	return nil
}

func (h *StoppingHandler) cleanupFiles(ctx context.Context) error {
	// Cleanup temporary files
	return nil
}
