# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    openssl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN make build

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    openssl \
    && update-ca-certificates

# Create non-root user
RUN addgroup -S sssonector && adduser -S sssonector -G sssonector

# Create necessary directories
RUN mkdir -p /etc/sssonector \
    /var/lib/sssonector \
    /var/log/sssonector \
    && chown -R sssonector:sssonector \
        /etc/sssonector \
        /var/lib/sssonector \
        /var/log/sssonector \
    && chmod -R 755 \
        /etc/sssonector \
        /var/lib/sssonector \
        /var/log/sssonector

# Copy binaries from builder
COPY --from=builder /build/dist/sssonector /usr/local/bin/
COPY --from=builder /build/dist/sssonectorctl /usr/local/bin/

# Copy configuration and security policies
COPY --chown=sssonector:sssonector config/ /etc/sssonector/
COPY --chown=sssonector:sssonector security/ /etc/sssonector/security/

# Set permissions
RUN chmod 755 /usr/local/bin/sssonector \
    /usr/local/bin/sssonectorctl

# Switch to non-root user
USER sssonector

# Expose ports
EXPOSE 8080 8443

# Set environment variables
ENV SSSONECTOR_CONFIG=/etc/sssonector/config.json \
    SSSONECTOR_LOG_LEVEL=info \
    SSSONECTOR_LOG_FORMAT=json

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD sssonectorctl health || exit 1

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/sssonector"]

# Default command
CMD ["serve"]

# Labels
LABEL org.opencontainers.image.title="SSSonector" \
    org.opencontainers.image.description="Enterprise-grade communications utility" \
    org.opencontainers.image.vendor="o3willard-AI" \
    org.opencontainers.image.source="https://github.com/o3willard-AI/SSSonector" \
    org.opencontainers.image.licenses="MIT"
