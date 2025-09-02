# Multi-stage Dockerfile for gomdlint
# Based on Go 1.24+ best practices and GEICO corporate security standards

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    curl \
    && update-ca-certificates

# Create non-root user for build
RUN adduser -D -g '' appuser

# Set build arguments for version injection
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG SERVICE=gomdlint

# GEICO Corporate Environment Setup
# Copy GEICO CA certificates (if available in build context)
COPY certs/ /usr/local/share/ca-certificates/ 2>/dev/null || echo "No GEICO certificates found in build context"
RUN update-ca-certificates

# Set GEICO-specific Go environment
ENV GOPROXY=https://artifactory-pd-infra.aks.aze1.cloud.geico.net/artifactory/api/go/mvp-billing-golang-all
ENV GONOSUMDB=github.com/geico-private/*
ENV GOSUMDB=sum.golang.org
ENV GEICO_ENV=container

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Test GEICO proxy connectivity and download dependencies
RUN echo "Testing GEICO Go proxy connectivity..." && \
    (curl -f --connect-timeout 10 "$GOPROXY" || echo "GEICO proxy not accessible, falling back to public proxy") && \
    go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with version injection and optimizations
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
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
      org.opencontainers.image.source="https://github.com/gomdlint/gomdlint" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="gomdlint" \
      maintainer="gomdlint team"

# Use non-root user
USER appuser

# Set entrypoint
ENTRYPOINT ["/gomdlint"]

# Default command
CMD ["--help"]
