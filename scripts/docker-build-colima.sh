#!/bin/bash
# Docker build script for Colima with Zscaler/proxy support
# Handles 127.0.0.1 proxy resolution and SSL certificate issues

set -euo pipefail

echo "🐳 Docker build script for Colima with proxy support"
echo "=============================================="

# Check if we're running on Colima
if ! docker context ls | grep -q colima; then
    echo "⚠️  Warning: Not running on Colima context"
    echo "   This script is optimized for Colima networking"
fi

# Detect host IP for proxy resolution
echo "🔍 Detecting host IP for Colima networking..."

# Get Colima VM IP range  
COLIMA_IP=$(colima ssh -- ip route show default 2>/dev/null | awk '/default/ { print $3 }' || echo "192.168.5.2")
echo "📡 Colima host IP detected: $COLIMA_IP"

# Update proxy URLs if they use 127.0.0.1
DOCKER_HTTP_PROXY="$HTTP_PROXY"
DOCKER_HTTPS_PROXY="$HTTPS_PROXY"

if [[ -n "$HTTP_PROXY" ]] && echo "$HTTP_PROXY" | grep -q "127.0.0.1"; then
    DOCKER_HTTP_PROXY=$(echo "$HTTP_PROXY" | sed "s/127.0.0.1/$COLIMA_IP/g")
    echo "🔄 Updated HTTP_PROXY: $HTTP_PROXY -> $DOCKER_HTTP_PROXY"
fi

if [[ -n "$HTTPS_PROXY" ]] && echo "$HTTPS_PROXY" | grep -q "127.0.0.1"; then
    DOCKER_HTTPS_PROXY=$(echo "$HTTPS_PROXY" | sed "s/127.0.0.1/$COLIMA_IP/g")
    echo "🔄 Updated HTTPS_PROXY: $HTTPS_PROXY -> $DOCKER_HTTPS_PROXY"
fi

# Check for Zscaler certificates
echo "🔐 Checking for Zscaler certificates..."
if ls certs/zscaler*.crt 2>/dev/null || ls certs/ZscalerRootCertificate*.crt 2>/dev/null; then
    echo "✅ Zscaler certificates found in certs/ directory"
else
    echo "⚠️  No Zscaler certificates found"
    echo "   If you're behind Zscaler, add certificates to certs/ directory"
    echo "   Common locations:"
    echo "     - /usr/local/share/ca-certificates/ZscalerRootCertificate*.crt"
    echo "     - ~/Library/Application Support/ZScaler/cert.pem"
fi

# Build arguments
BUILD_ARGS=(
    "--build-arg" "HTTP_PROXY=$DOCKER_HTTP_PROXY"
    "--build-arg" "HTTPS_PROXY=$DOCKER_HTTPS_PROXY"
    "--build-arg" "NO_PROXY=$NO_PROXY"
    "--build-arg" "VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo 'dev')"
    "--build-arg" "COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
    "--build-arg" "BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
)

# Image name
IMAGE_NAME="${1:-gomdlint}"

echo "🔨 Building Docker image..."
echo "   Image: $IMAGE_NAME"
echo "   Proxy: $DOCKER_HTTP_PROXY"
echo "   Build args: ${BUILD_ARGS[*]}"

# Build with timeout and better error handling
timeout 300 docker build "${BUILD_ARGS[@]}" -t "$IMAGE_NAME" . || {
    echo "❌ Docker build failed"
    echo ""
    echo "Common solutions for Colima + Zscaler:"
    echo "1. Check proxy connectivity: curl -v $DOCKER_HTTP_PROXY"
    echo "2. Verify Zscaler certificates in certs/"
    echo "3. Test Docker daemon proxy: docker info | grep -i proxy"
    echo "4. Restart Colima with proxy: colima restart --http-proxy=$HTTP_PROXY --https-proxy=$HTTPS_PROXY"
    echo ""
    exit 1
}

echo "✅ Docker build completed successfully"
echo "   Image: $IMAGE_NAME"
echo ""
echo "🧪 Testing the built image..."
docker run --rm "$IMAGE_NAME" version || echo "⚠️  Version test failed"

echo "✅ Build and test completed"
