# Build stage
FROM golang:1.26-alpine AS builder
# GOTOOLCHAIN=local forces an exact toolchain match: air-gapped builds fail
# fast instead of silently fetching a toolchain over the network.
ENV GOTOOLCHAIN=local

# Install build dependencies
RUN apk add --no-cache gcc musl-dev linux-headers

# Set working directory
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

ARG VERSION=dev

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -X main.Version=${VERSION}" -o sssonector ./cmd/daemon

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates iptables ip6tables && \
    mkdir -p /etc/sssonector /var/log/sssonector

# Copy binaries from builder
COPY --from=builder /build/sssonector /usr/local/bin/

# Make binaries executable
RUN chmod +x /usr/local/bin/sssonector

# Create non-root user
RUN adduser -D -H -s /sbin/nologin sssonector && \
    chown -R sssonector:sssonector /etc/sssonector /var/log/sssonector

# Set capabilities
RUN apk add --no-cache libcap && \
    setcap cap_net_admin,cap_net_raw+ep /usr/local/bin/sssonector && \
    apk del libcap

# Switch to non-root user
USER sssonector

# Expose ports
EXPOSE 443/tcp
EXPOSE 443/udp


# Set entrypoint
ENTRYPOINT ["/usr/local/bin/sssonector"]
CMD ["-config", "/etc/sssonector/config.yaml"]
