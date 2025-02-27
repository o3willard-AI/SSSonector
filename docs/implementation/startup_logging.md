# Startup Logging Implementation

## Overview

The startup logging system provides detailed insights into the SSSonector startup process, enabling better monitoring, debugging, and performance analysis. This document describes the implementation details, configuration options, and best practices.

## Architecture

### Core Components

1. **StartupLogger**
   - Manages logging phases
   - Tracks operation timing
   - Monitors resource states
   - Provides structured logging
   - Includes version information

2. **Configuration**
   - Integrated with LoggingConfig
   - Configurable through YAML
   - Default settings provided
   - Validation included
   - File output support

3. **Type System**
   - Centralized type definitions
   - Phase tracking enums
   - Component identification
   - Structured log entries
   - Version metadata

## Implementation Details

### Startup Phases

1. **PreStartup Phase**
   - Configuration validation
   - Resource availability checks
   - Environment verification
   - Permission validation
   - Version information logging

2. **Initialization Phase**
   - Memory allocation
   - Security context setup
   - Certificate loading
   - Network namespace creation

3. **Connection Phase**
   - Network interface setup
   - TLS handshake
   - Tunnel establishment
   - Route configuration

4. **Listen Phase (Server Only)**
   - Socket binding
   - Connection acceptance
   - Resource monitoring
   - Health checks

### Logging Structure

```go
type StartupLog struct {
    Phase     StartupPhase           `json:"phase" yaml:"phase"`
    Component StartupComponent       `json:"component" yaml:"component"`
    Operation string                 `json:"operation" yaml:"operation"`
    Details   map[string]interface{} `json:"details,omitempty" yaml:"details,omitempty"`
    Duration  Duration               `json:"duration,omitempty" yaml:"duration,omitempty"`
    Status    string                 `json:"status" yaml:"status"`
    Error     string                 `json:"error,omitempty" yaml:"error,omitempty"`
    Timestamp time.Time              `json:"timestamp" yaml:"timestamp"`
    Version   string                 `json:"version,omitempty" yaml:"version,omitempty"`
    BuildInfo map[string]string      `json:"build_info,omitempty" yaml:"build_info,omitempty"`
}
```

### Configuration Options

```yaml
logging:
  level: info        # debug, info, warn, error
  format: json       # json or console
  output: file       # file or stdout
  file: /path/to/logfile.log  # Required when output is file
  startup_logs: true  # Enable/disable startup logging
```

### Version Information

The startup logger includes version information in logs:
- Version number (from git tag)
- Build timestamp
- Git commit hash

This information is set during build time using ldflags:
```bash
go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.CommitHash=$COMMIT_HASH"
```

### Build System

A build script (`build.sh`) is provided to create binaries for all supported platforms:
- Linux (amd64, arm64, arm)
- macOS (amd64, arm64)
- Windows (amd64)

Each build includes:
- Static linking (CGO_ENABLED=0)
- Version information
- SHA256 checksums
- Platform-specific naming

## Usage Examples

### Version Information
```bash
$ sssonector --version
SSSonector v2.0.0-82-ge5bd185
Build Time: 2025-02-23_08:53:53
Commit: e5bd185
```

### Basic Implementation

```go
// Create startup logger
startupLogger := startup.NewStartupLogger(logger, config.Config.Logging)

// Set phase
startupLogger.SetPhase(types.StartupPhaseInitialization)

// Log operation with timing
err := startupLogger.LogOperation(
    types.StartupComponentAdapter,
    "Create TUN adapter",
    func() error {
        return createAdapter()
    },
    map[string]interface{}{
        "interface": "tun0",
        "mtu": 1500,
    },
)

// Log resource state
startupLogger.LogResourceState("adapter", map[string]interface{}{
    "state": "ready",
    "uptime": "30s",
})

// Log checkpoint
startupLogger.LogCheckpoint("Initialization complete", map[string]interface{}{
    "components": []string{"adapter", "security", "network"},
})
```

## Error Handling

1. **Operation Failures**
   - Detailed error logging
   - Stack trace inclusion
   - Context preservation
   - Cleanup handling

2. **Resource States**
   - State transition tracking
   - Error state detection
   - Recovery attempts
   - Cleanup procedures

## Performance Considerations

1. **Log Volume**
   - Configurable detail level
   - Efficient JSON encoding
   - Buffer management
   - Disk I/O optimization
   - File rotation support

2. **Memory Usage**
   - Object pooling
   - Buffer reuse
   - Garbage collection impact
   - Memory pressure monitoring

## Testing

1. **Unit Tests**
   - Phase transitions
   - Operation timing
   - Error handling
   - Format validation
   - Version information

2. **Integration Tests**
   - Full startup sequence
   - Resource management
   - Error scenarios
   - Performance impact
   - Build verification

3. **QA Testing**
   - Automated validation
   - Format verification
   - Performance benchmarks
   - Resource monitoring
   - Cross-platform testing

## Best Practices

1. **Configuration**
   - Enable startup logging in production
   - Use JSON format for structured analysis
   - Set appropriate log levels
   - Monitor log volume
   - Configure log file paths

2. **Implementation**
   - Log meaningful operations
   - Include relevant context
   - Handle errors appropriately
   - Clean up resources
   - Include version info

3. **Monitoring**
   - Track startup duration
   - Monitor resource states
   - Alert on failures
   - Analyze trends
   - Version tracking

4. **Maintenance**
   - Regular log rotation
   - Performance monitoring
   - Resource cleanup
   - Configuration updates
   - Version updates

## Future Improvements

1. **Enhanced Metrics**
   - Detailed timing analysis
   - Resource usage tracking
   - Performance correlation
   - Trend analysis
   - Version impact analysis

2. **Advanced Features**
   - Startup profiling
   - Automated analysis
   - Pattern detection
   - Predictive analytics
   - Version compatibility checks

3. **Integration**
   - Monitoring systems
   - Analytics platforms
   - Alert systems
   - Management tools
   - Version management systems
