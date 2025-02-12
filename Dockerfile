# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache make git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' sssonector

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/sssonector /app/
COPY --from=builder /app/bin/benchmark /app/

# Copy configuration files
COPY --from=builder /app/config /app/config

# Set ownership
RUN chown -R sssonector:sssonector /app

# Switch to non-root user
USER sssonector

# Expose ports
EXPOSE 8080

# Set environment variables
ENV GO_ENV=production

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["/app/sssonector"]

# Alternative commands available:
# - Run benchmarks: /app/benchmark
# Usage examples in README.md
