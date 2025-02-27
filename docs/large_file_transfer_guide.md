# SSSonector Large File Transfer Guide

This guide provides detailed information about transferring large files through SSSonector, including configuration options, performance considerations, and best practices.

## Overview

SSSonector supports transferring large files between endpoints connected through the tunnel. This capability is essential for various use cases, such as:

- Transferring database backups
- Sharing large media files
- Distributing software updates
- Synchronizing large datasets
- Backing up critical data

This guide will help you configure SSSonector for optimal large file transfer performance and reliability.

## Configuration Options

### Buffer Size

- **Option**: `tunnel.transfer.buffer_size`
- **Description**: Specifies the size of the buffer used for data transfer operations. A larger buffer can improve throughput for large file transfers but may increase memory usage.
- **Values**: Integer greater than 0, in bytes
- **Default**: `65536` (64 KB)
- **Example**: `tunnel.transfer.buffer_size: 262144` (256 KB)
- **Notes**:
  - For large file transfers, a larger buffer size can significantly improve performance.
  - However, a buffer that is too large may cause memory pressure, especially on systems with limited resources.
  - The optimal buffer size depends on the available memory, network conditions, and file size.

### Chunk Size

- **Option**: `tunnel.transfer.chunk_size`
- **Description**: Specifies the size of chunks used for large file transfers. Files larger than this size will be split into multiple chunks for transfer.
- **Values**: Integer greater than 0, in bytes
- **Default**: `1048576` (1 MB)
- **Example**: `tunnel.transfer.chunk_size: 4194304` (4 MB)
- **Notes**:
  - A larger chunk size can improve throughput for large file transfers but may increase memory usage.
  - A smaller chunk size can reduce memory usage but may decrease throughput.
  - The optimal chunk size depends on the available memory, network conditions, and file size.

### Parallel Transfers

- **Option**: `tunnel.transfer.parallel_transfers`
- **Description**: Specifies the maximum number of parallel transfers allowed. This option can improve throughput for large file transfers by utilizing available bandwidth more effectively.
- **Values**: Integer greater than 0
- **Default**: `4`
- **Example**: `tunnel.transfer.parallel_transfers: 8`
- **Notes**:
  - A higher number of parallel transfers can improve throughput for large file transfers but may increase CPU and memory usage.
  - A lower number of parallel transfers can reduce CPU and memory usage but may decrease throughput.
  - The optimal number of parallel transfers depends on the available CPU cores, memory, and network bandwidth.

### Compression

- **Option**: `tunnel.transfer.compression_enabled`
- **Description**: Specifies whether compression is enabled for data transfers. Compression can reduce the amount of data transferred over the network, improving throughput for compressible data.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `tunnel.transfer.compression_enabled: true`
- **Notes**:
  - Compression can significantly improve throughput for text-based files, such as logs, documents, and source code.
  - Compression may not improve throughput for already-compressed files, such as images, videos, and archives.
  - Compression requires additional CPU resources, which may impact performance on CPU-constrained systems.

### Compression Level

- **Option**: `tunnel.transfer.compression_level`
- **Description**: Specifies the compression level to use when compression is enabled. A higher compression level can achieve better compression ratios but requires more CPU resources.
- **Values**: Integer from 1 to 9, where 1 is the fastest and 9 is the most compressed
- **Default**: `6`
- **Example**: `tunnel.transfer.compression_level: 4`
- **Notes**:
  - A lower compression level (1-3) is faster but achieves lower compression ratios.
  - A medium compression level (4-6) provides a good balance between speed and compression ratio.
  - A higher compression level (7-9) achieves better compression ratios but is slower.
  - The optimal compression level depends on the available CPU resources, network bandwidth, and file type.

### Checksum Verification

- **Option**: `tunnel.transfer.checksum_verification`
- **Description**: Specifies whether checksum verification is enabled for data transfers. Checksum verification ensures data integrity by verifying that the data received matches the data sent.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `tunnel.transfer.checksum_verification: true`
- **Notes**:
  - Checksum verification can detect data corruption during transfer, ensuring data integrity.
  - Checksum verification requires additional CPU resources, which may impact performance on CPU-constrained systems.
  - For large file transfers, checksum verification is especially important to ensure data integrity.

### Retry Mechanisms

- **Option**: `tunnel.transfer.retry_enabled`
- **Description**: Specifies whether retry mechanisms are enabled for data transfers. Retry mechanisms can improve reliability by automatically retrying failed transfers.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `tunnel.transfer.retry_enabled: true`
- **Notes**:
  - Retry mechanisms are especially important for large file transfers, which are more likely to encounter transient network issues.
  - For more information on retry mechanisms, see the [Error Handling Guide](error_handling_guide.md).

## Performance Considerations

### Network Bandwidth

Network bandwidth is a critical factor in large file transfer performance. The maximum throughput achievable is limited by the available bandwidth between the endpoints.

To optimize network bandwidth utilization:

1. **Minimize Competing Traffic**: Reduce other network traffic during large file transfers to maximize available bandwidth.
2. **Use Wired Connections**: Wired connections typically provide more stable and higher bandwidth than wireless connections.
3. **Consider Time of Day**: Network congestion can vary throughout the day. Schedule large file transfers during off-peak hours when possible.
4. **Monitor Bandwidth Usage**: Use network monitoring tools to track bandwidth usage and identify bottlenecks.

### Latency

Network latency can significantly impact large file transfer performance, especially for protocols that require frequent acknowledgments.

To mitigate the impact of latency:

1. **Increase Buffer Size**: A larger buffer size can help compensate for high latency by reducing the frequency of acknowledgments.
2. **Use Parallel Transfers**: Parallel transfers can help utilize available bandwidth more effectively in high-latency environments.
3. **Consider Geographical Distance**: Transfers between geographically distant endpoints will experience higher latency. Adjust expectations and configurations accordingly.

### CPU Resources

Large file transfers can be CPU-intensive, especially when compression and encryption are enabled.

To optimize CPU resource utilization:

1. **Adjust Compression Level**: Lower the compression level if CPU usage is a concern.
2. **Monitor CPU Usage**: Use system monitoring tools to track CPU usage during transfers and adjust configurations if necessary.
3. **Consider Hardware Acceleration**: Some systems support hardware acceleration for encryption, which can reduce CPU load.

### Memory Resources

Large file transfers can consume significant memory, especially with large buffer and chunk sizes.

To optimize memory resource utilization:

1. **Adjust Buffer Size**: Reduce the buffer size if memory usage is a concern.
2. **Adjust Chunk Size**: Reduce the chunk size if memory usage is a concern.
3. **Monitor Memory Usage**: Use system monitoring tools to track memory usage during transfers and adjust configurations if necessary.

## Common Use Cases

### Transferring Database Backups

Database backups are typically large, compressed files that need to be transferred securely and reliably.

Recommended configuration:

```yaml
tunnel:
  transfer:
    buffer_size: 262144        # 256 KB
    chunk_size: 4194304        # 4 MB
    parallel_transfers: 4
    compression_enabled: false # Database backups are usually already compressed
    checksum_verification: true
    retry_enabled: true
```

### Sharing Large Media Files

Media files, such as videos and high-resolution images, are typically large and already compressed.

Recommended configuration:

```yaml
tunnel:
  transfer:
    buffer_size: 524288        # 512 KB
    chunk_size: 8388608        # 8 MB
    parallel_transfers: 8
    compression_enabled: false # Media files are usually already compressed
    checksum_verification: true
    retry_enabled: true
```

### Distributing Software Updates

Software updates often include a mix of compressed and uncompressed files that need to be distributed to multiple endpoints.

Recommended configuration:

```yaml
tunnel:
  transfer:
    buffer_size: 262144        # 256 KB
    chunk_size: 4194304        # 4 MB
    parallel_transfers: 4
    compression_enabled: true  # Effective for mixed content
    compression_level: 4       # Balance between speed and compression
    checksum_verification: true
    retry_enabled: true
```

### Synchronizing Large Datasets

Large datasets often include a mix of file types and sizes that need to be synchronized between endpoints.

Recommended configuration:

```yaml
tunnel:
  transfer:
    buffer_size: 262144        # 256 KB
    chunk_size: 4194304        # 4 MB
    parallel_transfers: 4
    compression_enabled: true  # Effective for mixed content
    compression_level: 6       # Default level
    checksum_verification: true
    retry_enabled: true
```

## Troubleshooting

### Slow Transfer Speeds

If you're experiencing slow transfer speeds, consider the following troubleshooting steps:

1. **Check Network Bandwidth**: Verify that sufficient bandwidth is available for the transfer.
2. **Monitor CPU Usage**: High CPU usage can limit transfer speeds. Consider reducing the compression level or disabling compression.
3. **Monitor Memory Usage**: High memory usage can limit transfer speeds. Consider reducing the buffer size or chunk size.
4. **Check for Network Congestion**: Network congestion can limit transfer speeds. Consider scheduling transfers during off-peak hours.
5. **Verify MTU Settings**: Incorrect MTU settings can cause fragmentation and reduce transfer speeds. See the [Advanced Configuration Guide](advanced_configuration_guide.md) for more information on MTU settings.
6. **Adjust Parallel Transfers**: Increasing the number of parallel transfers can improve throughput in some cases.

### Transfer Failures

If transfers are failing, consider the following troubleshooting steps:

1. **Check Network Connectivity**: Verify that network connectivity is stable between the endpoints.
2. **Check Disk Space**: Verify that sufficient disk space is available on the receiving endpoint.
3. **Check File Permissions**: Verify that the necessary file permissions are available on both endpoints.
4. **Enable Retry Mechanisms**: Ensure that retry mechanisms are enabled to handle transient network issues.
5. **Check Logs**: Review the logs for error messages that might indicate the cause of the failure.
6. **Verify Checksum Verification**: Ensure that checksum verification is enabled to detect data corruption.

### Memory Issues

If you're experiencing memory issues during large file transfers, consider the following troubleshooting steps:

1. **Reduce Buffer Size**: A smaller buffer size can reduce memory usage.
2. **Reduce Chunk Size**: A smaller chunk size can reduce memory usage.
3. **Reduce Parallel Transfers**: Fewer parallel transfers can reduce memory usage.
4. **Monitor Memory Usage**: Use system monitoring tools to track memory usage during transfers and identify patterns.
5. **Consider System Limits**: Verify that system limits, such as maximum file descriptors, are not being exceeded.

## Best Practices

### Configuration

1. **Start with Default Values**: Begin with the default configuration values and adjust as needed based on performance and resource utilization.
2. **Test Different Configurations**: Experiment with different configuration values to find the optimal settings for your specific use case.
3. **Consider File Types**: Adjust compression settings based on the types of files being transferred.
4. **Balance Resources**: Find the right balance between CPU, memory, and network resource utilization.
5. **Document Configurations**: Document the configurations that work best for different use cases for future reference.

### Monitoring

1. **Monitor Transfer Speeds**: Track transfer speeds to identify performance trends and issues.
2. **Monitor Resource Utilization**: Track CPU, memory, and network resource utilization during transfers.
3. **Monitor Disk Space**: Ensure that sufficient disk space is available on the receiving endpoint.
4. **Set Up Alerts**: Configure alerts for abnormal conditions, such as failed transfers or resource exhaustion.
5. **Review Logs Regularly**: Regularly review logs to identify patterns and potential issues.

### Security

1. **Enable TLS**: Ensure that TLS is enabled to encrypt data in transit.
2. **Verify Certificates**: Verify that certificates are valid and trusted.
3. **Use Mutual Authentication**: Consider using mutual TLS authentication for additional security.
4. **Restrict Access**: Limit access to the tunnel to authorized users and systems.
5. **Audit Transfers**: Maintain audit logs of file transfers for security and compliance purposes.

### Reliability

1. **Enable Checksum Verification**: Ensure that checksum verification is enabled to detect data corruption.
2. **Enable Retry Mechanisms**: Ensure that retry mechanisms are enabled to handle transient network issues.
3. **Implement Monitoring**: Set up monitoring to detect and alert on transfer failures.
4. **Test Regularly**: Regularly test file transfers to ensure that they are working as expected.
5. **Have Backup Methods**: Have alternative methods for transferring files in case of persistent issues.

## Example Configurations

### Balanced Configuration

This configuration provides a good balance between performance and resource utilization for most use cases:

```yaml
tunnel:
  transfer:
    buffer_size: 262144        # 256 KB
    chunk_size: 4194304        # 4 MB
    parallel_transfers: 4
    compression_enabled: true
    compression_level: 6
    checksum_verification: true
    retry_enabled: true
```

### High-Performance Configuration

This configuration prioritizes performance over resource utilization, suitable for systems with ample resources:

```yaml
tunnel:
  transfer:
    buffer_size: 1048576       # 1 MB
    chunk_size: 16777216       # 16 MB
    parallel_transfers: 8
    compression_enabled: true
    compression_level: 4       # Balance between speed and compression
    checksum_verification: true
    retry_enabled: true
```

### Low-Resource Configuration

This configuration prioritizes resource utilization over performance, suitable for systems with limited resources:

```yaml
tunnel:
  transfer:
    buffer_size: 65536         # 64 KB
    chunk_size: 1048576        # 1 MB
    parallel_transfers: 2
    compression_enabled: true
    compression_level: 6
    checksum_verification: true
    retry_enabled: true
```

### No-Compression Configuration

This configuration disables compression, suitable for transferring already-compressed files:

```yaml
tunnel:
  transfer:
    buffer_size: 262144        # 256 KB
    chunk_size: 4194304        # 4 MB
    parallel_transfers: 4
    compression_enabled: false
    checksum_verification: true
    retry_enabled: true
```

## Environment Variables

SSSonector also supports configuration of large file transfer options through environment variables:

- `SSSONECTOR_TUNNEL_TRANSFER_BUFFER_SIZE`: Specifies the buffer size.
- `SSSONECTOR_TUNNEL_TRANSFER_CHUNK_SIZE`: Specifies the chunk size.
- `SSSONECTOR_TUNNEL_TRANSFER_PARALLEL_TRANSFERS`: Specifies the number of parallel transfers.
- `SSSONECTOR_TUNNEL_TRANSFER_COMPRESSION_ENABLED`: Specifies whether compression is enabled.
- `SSSONECTOR_TUNNEL_TRANSFER_COMPRESSION_LEVEL`: Specifies the compression level.
- `SSSONECTOR_TUNNEL_TRANSFER_CHECKSUM_VERIFICATION`: Specifies whether checksum verification is enabled.
- `SSSONECTOR_TUNNEL_TRANSFER_RETRY_ENABLED`: Specifies whether retry mechanisms are enabled.

Example usage:

```bash
export SSSONECTOR_TUNNEL_TRANSFER_BUFFER_SIZE=262144
export SSSONECTOR_TUNNEL_TRANSFER_CHUNK_SIZE=4194304
export SSSONECTOR_TUNNEL_TRANSFER_PARALLEL_TRANSFERS=4
export SSSONECTOR_TUNNEL_TRANSFER_COMPRESSION_ENABLED=true
export SSSONECTOR_TUNNEL_TRANSFER_COMPRESSION_LEVEL=6
export SSSONECTOR_TUNNEL_TRANSFER_CHECKSUM_VERIFICATION=true
export SSSONECTOR_TUNNEL_TRANSFER_RETRY_ENABLED=true

./sssonector
```

## Conclusion

SSSonector provides robust support for large file transfers, with configurable options to optimize performance and reliability. By understanding and properly configuring these options, you can achieve optimal transfer speeds while ensuring data integrity and security.

For more information on other configuration options, see the [Advanced Configuration Guide](advanced_configuration_guide.md).
