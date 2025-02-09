# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev \
    make \
    git

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN make build-linux-amd64

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    iptables \
    iproute2

# Create non-root user
RUN addgroup -S sssonector && \
    adduser -S -G sssonector sssonector

# Create necessary directories
RUN mkdir -p /etc/sssonector /var/lib/sssonector /var/run/sssonector && \
    chown -R sssonector:sssonector /etc/sssonector /var/lib/sssonector /var/run/sssonector

# Copy binaries from builder
COPY --from=builder /build/build/linux/amd64/sssonector /usr/local/bin/
COPY --from=builder /build/build/linux/amd64/sssonectorctl /usr/local/bin/

# Copy configuration and scripts
COPY config/default.json /etc/sssonector/config.json
COPY scripts/docker-entrypoint.sh /usr/local/bin/

# Set permissions
RUN chmod +x /usr/local/bin/sssonector /usr/local/bin/sssonectorctl /usr/local/bin/docker-entrypoint.sh && \
    chown root:sssonector /usr/local/bin/sssonector /usr/local/bin/sssonectorctl && \
    chmod 750 /usr/local/bin/sssonector /usr/local/bin/sssonectorctl

# Set capabilities
RUN setcap cap_net_admin,cap_net_bind_service,cap_net_raw+ep /usr/local/bin/sssonector

# Expose monitoring ports
EXPOSE 9090
EXPOSE 161

# Set working directory
WORKDIR /var/lib/sssonector

# Switch to non-root user
USER sssonector

# Set environment variables
ENV PATH="/usr/local/bin:${PATH}" \
    CONFIG_FILE="/etc/sssonector/config.json" \
    DATA_DIR="/var/lib/sssonector" \
    LOG_LEVEL="info"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["sssonectorctl", "health"]

# Entrypoint
ENTRYPOINT ["docker-entrypoint.sh"]

# Default command
CMD ["sssonector", "--config", "/etc/sssonector/config.json"]

# Labels
LABEL org.opencontainers.image.title="SSSonector" \
      org.opencontainers.image.description="High-performance communications utility" \
      org.opencontainers.image.vendor="o3willard-AI" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.url="https://github.com/o3willard-AI/SSSonector" \
      org.opencontainers.image.documentation="https://github.com/o3willard-AI/SSSonector/docs" \
      org.opencontainers.image.source="https://github.com/o3willard-AI/SSSonector" \
      org.opencontainers.image.licenses="MIT"
