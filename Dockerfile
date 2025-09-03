# Multi-stage Dockerfile for gomdlint
# Based on Go 1.23+ with proxy and Zscaler SSL support

# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies including certificate tools
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    curl \
    openssl \
    && update-ca-certificates

# Create non-root user for build
RUN adduser -D -g '' appuser

# Set build arguments for version injection
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG SERVICE=gomdlint

# Handle proxy configuration for Zscaler/corporate environments
# Use host.docker.internal for Docker Desktop or host.containers.internal for Podman
# For Colima, we need to use the host IP from the container perspective
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV NO_PROXY=${NO_PROXY}

# Copy custom certificates if available (for Zscaler/corporate environments)
# Create certificates directory first
RUN mkdir -p /usr/local/share/ca-certificates/

# Copy certificate directory if it exists, otherwise create empty directory
COPY certs /tmp/certs/
RUN if [ -d /tmp/certs ]; then \
    find /tmp/certs -name "*.crt" -exec cp {} /usr/local/share/ca-certificates/ \; || true; \
    find /tmp/certs -name "*.pem" -exec cp {} /usr/local/share/ca-certificates/ \; || true; \
    echo "Certificate installation completed"; \
    else echo "No certificates directory found"; fi

# Update certificate store
RUN update-ca-certificates

# Debug: List installed certificates for troubleshooting
RUN ls -la /usr/local/share/ca-certificates/ || echo "No custom certificates installed"

# Set Go environment with proxy fallback
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
ENV GOPRIVATE=""

# Handle Colima networking - 127.0.0.1 proxies need host IP resolution
RUN if [ -n "$HTTP_PROXY" ] && echo "$HTTP_PROXY" | grep -q "127.0.0.1"; then \
        echo "Detected 127.0.0.1 proxy, resolving host IP for Colima networking..."; \
        # Try multiple methods to find the host IP from inside Colima container \
        HOST_IP=$(getent hosts host.docker.internal | awk '{ print $1 }' 2>/dev/null || \
                  getent hosts gateway.docker.internal | awk '{ print $1 }' 2>/dev/null || \
                  ip route show default | awk '/default/ { print $3 }' 2>/dev/null || \
                  cat /proc/net/route | grep '^00000000' | awk '{print $2}' | \
                  while read hex_ip; do printf "%d.%d.%d.%d\n" 0x${hex_ip:6:2} 0x${hex_ip:4:2} 0x${hex_ip:2:2} 0x${hex_ip:0:2}; done | head -1 || \
                  echo "192.168.5.2"); \
        echo "Resolved Colima host IP: $HOST_IP"; \
        NEW_HTTP_PROXY=$(echo "$HTTP_PROXY" | sed "s/127.0.0.1/$HOST_IP/g"); \
        NEW_HTTPS_PROXY=$(echo "$HTTPS_PROXY" | sed "s/127.0.0.1/$HOST_IP/g"); \
        echo "Original HTTP_PROXY: $HTTP_PROXY"; \
        echo "Updated HTTP_PROXY: $NEW_HTTP_PROXY"; \
        export HTTP_PROXY="$NEW_HTTP_PROXY"; \
        export HTTPS_PROXY="$NEW_HTTPS_PROXY"; \
        # Test proxy connectivity \
        curl -s --connect-timeout 10 "$HTTP_PROXY" || echo "Proxy connectivity test failed, continuing..."; \
    else \
        echo "No 127.0.0.1 proxy detected or HTTP_PROXY not set"; \
    fi

# Test SSL connectivity with Zscaler/proxy setup
RUN echo "Testing SSL connectivity through proxy..." && \
    (curl -s --connect-timeout 10 https://proxy.golang.org || \
     echo "Direct HTTPS failed, will rely on proxy settings") && \
    echo "SSL connectivity test completed"

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with version injection and optimizations
# Build arguments for cross-compilation
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE} -X main.service=${SERVICE}" \
    -a \
    -installsuffix cgo \
    -o gomdlint \
    ./cmd/gomdlint

# Verify the binary
RUN ./gomdlint version

# Final stage - minimal runtime
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /build/gomdlint /gomdlint

# Add version labels
ARG VERSION
ARG COMMIT  
ARG BUILD_DATE
ARG SERVICE

LABEL org.opencontainers.image.title="gomdlint" \
      org.opencontainers.image.description="High-performance Go Markdown linter" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.service="${SERVICE}" \
      org.opencontainers.image.source="https://github.com/jamesainslie/gomdlint" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="jamesainslie" \
      maintainer="jamesainslie"

# Use non-root user
USER appuser

# Set entrypoint
ENTRYPOINT ["/gomdlint"]

# Default command
CMD ["--help"]
