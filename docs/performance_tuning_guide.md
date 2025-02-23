# SSSonector Performance Tuning Guide

## Overview
This guide provides detailed information about performance optimization, resource sizing, monitoring, and troubleshooting for SSSonector deployments.

## Table of Contents
1. [Resource Requirements](#resource-requirements)
2. [Performance Optimization](#performance-optimization)
3. [Configuration Tuning](#configuration-tuning)
4. [Monitoring and Metrics](#monitoring-and-metrics)
5. [Troubleshooting](#troubleshooting)
6. [Benchmarking](#benchmarking)

## Resource Requirements

### Minimum Requirements
- CPU: 2 cores
- Memory: 1GB
- Network: 100Mbps
- Disk: 1GB

### Recommended Requirements
- CPU: 4+ cores
- Memory: 4GB
- Network: 1Gbps
- Disk: 10GB

### Scaling Guidelines
```yaml
# Resource scaling per 1000 connections
resources:
  cpu_cores: 2
  memory_gb: 2
  network_mbps: 100
  disk_gb: 5
```

## Performance Optimization

### Connection Pooling
```yaml
pool:
  # Optimal pool size based on available memory
  initial_size: ${MEMORY_GB * 100}  # connections
  max_size: ${MEMORY_GB * 500}      # connections
  
  # Connection lifecycle
  idle_timeout: 300s
  max_lifetime: 3600s
  
  # Resource management
  cleanup_interval: 60s
  health_check_interval: 30s
```

### Buffer Management
```yaml
buffers:
  # Buffer sizing
  read_size: 32KB
  write_size: 32KB
  
  # Pool configuration
  pool_size: ${MAX_CONNECTIONS * 2}
  prealloc: true
  
  # Memory management
  max_memory: ${TOTAL_MEMORY * 0.3}  # 30% of total memory
  cleanup_interval: 30s
```

### Rate Limiting
```yaml
rate_limit:
  # Base configuration
  base_rate: ${NETWORK_MBPS * 10}  # requests/second
  burst_size: ${BASE_RATE * 0.1}   # 10% of base rate
  
  # Dynamic adjustment
  enable_dynamic: true
  min_rate: ${BASE_RATE * 0.5}     # 50% of base rate
  max_rate: ${BASE_RATE * 2.0}     # 200% of base rate
  
  # Cleanup
  cleanup_interval: 60s
```

## Configuration Tuning

### Network Tuning
```yaml
network:
  # TUN interface
  mtu: 1500
  tx_queue_len: 1000
  
  # TCP parameters
  tcp_keepalive: true
  tcp_keepalive_time: 60
  tcp_keepalive_intvl: 10
  tcp_keepalive_probes: 6
  
  # Buffer sizes
  rmem_default: 262144    # 256KB
  rmem_max: 4194304      # 4MB
  wmem_default: 262144    # 256KB
  wmem_max: 4194304      # 4MB
```

### System Tuning
```bash
# System limits
ulimit -n 65535              # File descriptors
ulimit -u 65535              # User processes
sysctl -w vm.swappiness=10   # Reduce swapping
```

### Process Tuning
```yaml
process:
  # Thread pool
  worker_threads: ${CPU_CORES * 2}
  io_threads: ${CPU_CORES}
  
  # Memory limits
  max_memory: ${TOTAL_MEMORY * 0.8}  # 80% of total memory
  
  # CPU affinity
  cpu_affinity: true
  numa_aware: true
```

## Monitoring and Metrics

### Key Metrics

1. **Connection Metrics**
   ```yaml
   metrics:
     connections:
       active: gauge
       idle: gauge
       total: counter
       errors: counter
       latency: histogram
   ```

2. **Resource Metrics**
   ```yaml
   metrics:
     resources:
       cpu_usage: gauge
       memory_usage: gauge
       network_in: counter
       network_out: counter
       disk_io: gauge
   ```

3. **Performance Metrics**
   ```yaml
   metrics:
     performance:
       request_rate: gauge
       error_rate: gauge
       latency_p95: gauge
       latency_p99: gauge
       throughput: gauge
   ```

### Prometheus Integration
```yaml
prometheus:
  enabled: true
  port: 9090
  path: /metrics
  labels:
    environment: production
    component: sssonector
```

### Grafana Dashboard
```yaml
grafana:
  datasource: prometheus
  refresh: 10s
  panels:
    - name: Connection Status
      metrics:
        - connections_active
        - connections_idle
        - connections_total
    - name: Resource Usage
      metrics:
        - cpu_usage
        - memory_usage
        - network_io
    - name: Performance
      metrics:
        - request_rate
        - latency_p95
        - throughput
```

## Troubleshooting

### Common Issues

1. **High Memory Usage**
   ```yaml
   symptoms:
     - Memory usage above 80%
     - Increased GC activity
   solutions:
     - Reduce connection pool size
     - Decrease buffer sizes
     - Enable memory limiting
     - Increase cleanup frequency
   ```

2. **High CPU Usage**
   ```yaml
   symptoms:
     - CPU usage above 70%
     - Increased latency
   solutions:
     - Reduce worker threads
     - Enable CPU affinity
     - Optimize rate limits
     - Scale horizontally
   ```

3. **Network Bottlenecks**
   ```yaml
   symptoms:
     - Increased latency
     - Reduced throughput
   solutions:
     - Optimize MTU size
     - Tune TCP parameters
     - Adjust buffer sizes
     - Check network capacity
   ```

### Performance Analysis

1. **CPU Profiling**
   ```bash
   # CPU profile collection
   go tool pprof http://localhost:6060/debug/pprof/profile
   
   # Analysis commands
   top10         # Show top 10 CPU consumers
   web          # Generate graph visualization
   list main    # Show annotated source
   ```

2. **Memory Profiling**
   ```bash
   # Memory profile collection
   go tool pprof http://localhost:6060/debug/pprof/heap
   
   # Analysis commands
   top          # Show memory consumers
   web          # Visualize allocations
   list         # Show allocation sites
   ```

3. **Goroutine Analysis**
   ```bash
   # Goroutine dump
   curl http://localhost:6060/debug/pprof/goroutine
   
   # Analysis
   go tool pprof -http=:8080 goroutine.prof
   ```

## Benchmarking

### Performance Testing
```bash
# Basic benchmark
go test -bench=. -benchmem ./...

# Detailed benchmark
go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./...
```

### Load Testing
```yaml
load_test:
  # Test configuration
  duration: 5m
  ramp_up: 30s
  
  # Load parameters
  connections: 1000
  requests_per_second: 5000
  payload_size: 1KB
  
  # Success criteria
  max_latency_p95: 100ms
  max_error_rate: 0.1%
  min_throughput: 50MB/s
```

### Stress Testing
```yaml
stress_test:
  # Test configuration
  duration: 1h
  max_connections: 10000
  
  # Resource limits
  cpu_limit: 80%
  memory_limit: 80%
  
  # Success criteria
  stability_period: 30m
  max_error_rate: 1%
  recovery_time: 5m
```

## Best Practices

### Resource Management
1. Monitor resource usage continuously
2. Set appropriate limits and thresholds
3. Implement automatic scaling
4. Regular performance testing

### Configuration Management
1. Version control configurations
2. Document all changes
3. Test in staging environment
4. Monitor impact of changes

### Performance Monitoring
1. Set up comprehensive monitoring
2. Define clear alerts
3. Regular performance reviews
4. Trend analysis

## Appendix

### System Commands
```bash
# System tuning
sysctl -w net.core.rmem_max=4194304
sysctl -w net.core.wmem_max=4194304
sysctl -w net.ipv4.tcp_rmem="4096 87380 4194304"
sysctl -w net.ipv4.tcp_wmem="4096 87380 4194304"

# Process limits
ulimit -n 65535
ulimit -u 65535

# Network tuning
ip link set dev tun0 txqueuelen 1000
ethtool -K tun0 tso on
ethtool -K tun0 gso on
```

### Configuration Templates
```yaml
# High-performance configuration
high_performance:
  pool:
    initial_size: 1000
    max_size: 5000
    idle_timeout: 300s
  
  buffers:
    size: 32KB
    pool_size: 10000
    prealloc: true
  
  network:
    mtu: 1500
    compression: false
    tcp_keepalive: true

# Resource-constrained configuration
resource_constrained:
  pool:
    initial_size: 100
    max_size: 500
    idle_timeout: 60s
  
  buffers:
    size: 16KB
    pool_size: 1000
    prealloc: false
  
  network:
    mtu: 1400
    compression: true
    tcp_keepalive: false
```

### Performance Checklist
- [ ] Resource requirements verified
- [ ] System tuning applied
- [ ] Configuration optimized
- [ ] Monitoring configured
- [ ] Benchmarks executed
- [ ] Performance validated
- [ ] Documentation updated
