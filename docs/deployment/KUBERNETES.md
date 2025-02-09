# SSSonector Kubernetes Deployment Guide

## Overview

SSSonector can be deployed on Kubernetes using a DaemonSet to ensure one instance runs on each node. This guide covers the requirements, limitations, and configuration details for running SSSonector in a Kubernetes environment.

## Requirements

### Node Requirements

1. Linux nodes with:
   - Kernel 4.9 or later
   - TUN/TAP support enabled
   - NET_ADMIN and NET_RAW capabilities
   - Access to /dev, /sys, and /proc

2. Container Runtime:
   - Docker 19.03+ or containerd 1.4+
   - Privileged container support
   - Host network access

### Kubernetes Requirements

1. Version Requirements:
   - Kubernetes 1.19 or later
   - CNI plugin with support for host networking
   - RBAC enabled

2. Cluster Permissions:
   - Ability to run privileged containers
   - Access to host network namespace
   - Permission to create network devices

## Limitations

1. Network Constraints:
   - Must run with host network (`hostNetwork: true`)
   - Cannot use standard Kubernetes networking
   - Limited CNI plugin compatibility

2. Security Implications:
   - Requires privileged container access
   - Needs host network access
   - Requires root privileges

3. Scalability Considerations:
   - One instance per node (DaemonSet)
   - Cannot horizontally scale on same node
   - Resource limits tied to node capacity

## Deployment Configuration

### 1. Security Context

```yaml
securityContext:
  privileged: true
  capabilities:
    add:
      - NET_ADMIN
      - NET_RAW
      - SYS_ADMIN
```

### 2. Network Configuration

```yaml
spec:
  hostNetwork: true
  dnsPolicy: ClusterFirstWithHostNet
```

### 3. Volume Mounts

```yaml
volumeMounts:
  - name: host-dev
    mountPath: /dev
  - name: host-sys
    mountPath: /sys
  - name: host-proc
    mountPath: /host/proc
```

## Installation Steps

1. Create namespace:
```bash
kubectl create namespace sssonector
```

2. Apply RBAC configuration:
```bash
kubectl apply -f deploy/kubernetes/rbac.yaml
```

3. Create ConfigMap and Secrets:
```bash
# Create certificates
./scripts/generate-certs.sh

# Create ConfigMap
kubectl create configmap sssonector-config --from-file=config.json

# Create Secrets
kubectl create secret tls sssonector-certs \
  --cert=certs/server.crt \
  --key=certs/server.key
```

4. Deploy SSSonector:
```bash
kubectl apply -f deploy/kubernetes/sssonector.yaml
```

## Verification

1. Check DaemonSet status:
```bash
kubectl get daemonset sssonector
```

2. Verify pod status:
```bash
kubectl get pods -l app=sssonector
```

3. Check logs:
```bash
kubectl logs -l app=sssonector
```

## Monitoring

1. Prometheus metrics available at:
   - http://[node-ip]:8080/metrics

2. Default metrics include:
   - Connection counts
   - Bandwidth usage
   - Error rates
   - Resource utilization

## Troubleshooting

### 1. Pod Startup Issues

Check for:
- Missing capabilities
- Volume mount permissions
- Host network access

```bash
kubectl describe pod -l app=sssonector
```

### 2. Network Issues

Verify:
- TUN/TAP device creation
- Network interface configuration
- Routing table updates

```bash
# Check pod network configuration
kubectl exec -it [pod-name] -- ip addr

# Verify TUN interface
kubectl exec -it [pod-name] -- ip link show tun0
```

### 3. Permission Issues

Check:
- RBAC configuration
- Security context
- Host path permissions

```bash
# View pod security context
kubectl get pod [pod-name] -o yaml
```

## Best Practices

1. Security:
   - Use network policies
   - Implement pod security policies
   - Regular certificate rotation
   - Audit logging

2. Monitoring:
   - Set up alerts
   - Monitor resource usage
   - Track network metrics
   - Log aggregation

3. Maintenance:
   - Regular updates
   - Configuration backups
   - Certificate management
   - Health checks

## Alternative Deployment Options

If Kubernetes deployment constraints are too restrictive, consider:

1. Bare metal deployment
2. Docker Compose deployment
3. Virtual machine deployment

These alternatives might provide more flexibility in network configuration and security requirements.

## Support

For issues specific to Kubernetes deployment:
1. Check the troubleshooting guide
2. Review Kubernetes logs
3. Contact support with:
   - Pod descriptions
   - Node information
   - Network configuration
   - Error logs
