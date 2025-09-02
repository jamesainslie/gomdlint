#!/bin/bash
# Setup script for Colima with Zscaler proxy support
# Configures Colima to work properly with corporate proxies

set -euo pipefail

echo "🚀 Setting up Colima for Zscaler/corporate proxy environments"
echo "============================================================"

# Check if Colima is installed
if ! command -v colima >/dev/null 2>&1; then
    echo "❌ Colima not installed. Install with: brew install colima"
    exit 1
fi

# Check current proxy settings
echo "🔍 Current proxy configuration:"
echo "   HTTP_PROXY: ${HTTP_PROXY:-not set}"
echo "   HTTPS_PROXY: ${HTTPS_PROXY:-not set}"
echo "   NO_PROXY: ${NO_PROXY:-not set}"
NO_PROXY="${NO_PROXY:-localhost,127.0.0.1,*.local}"

if [[ -z "${HTTP_PROXY:-}" ]]; then
    echo "⚠️  No proxy detected. If you're behind a corporate proxy:"
    echo "   export HTTP_PROXY=http://127.0.0.1:9000"
    echo "   export HTTPS_PROXY=http://127.0.0.1:9000"
    exit 0
fi

# Extract proxy host and port
PROXY_HOST=$(echo "$HTTP_PROXY" | sed -E 's|https?://([^:]+):.*|\1|')
PROXY_PORT=$(echo "$HTTP_PROXY" | sed -E 's|https?://[^:]+:([0-9]+).*|\1|')

echo "📡 Proxy details:"
echo "   Host: $PROXY_HOST"
echo "   Port: $PROXY_PORT"

# Check if Colima is running and stop it
if colima status >/dev/null 2>&1; then
    echo "🛑 Stopping Colima to reconfigure..."
    colima stop
fi

# Start Colima with proxy configuration
echo "🚀 Starting Colima with proxy configuration..."

# Start Colima normally first, then configure proxy
colima start --cpu 4 --memory 8 --disk 60 || {
    echo "❌ Colima start failed"
    exit 1
}

echo "🔧 Configuring Docker daemon proxy settings..."

# Configure Docker daemon proxy through Colima SSH
colima ssh -- sudo mkdir -p /etc/systemd/system/docker.service.d
colima ssh -- "echo '[Service]
Environment=\"HTTP_PROXY=$HTTP_PROXY\"
Environment=\"HTTPS_PROXY=$HTTPS_PROXY\"
Environment=\"NO_PROXY=$NO_PROXY\"' | sudo tee /etc/systemd/system/docker.service.d/http-proxy.conf"

echo "♻️  Reloading Docker daemon..."
colima ssh -- sudo systemctl daemon-reload
colima ssh -- sudo systemctl restart docker

# Wait for Docker to restart
sleep 5

echo "✅ Colima configured with proxy settings"

# Test Docker connectivity
echo "🧪 Testing Docker connectivity..."
if docker info >/dev/null 2>&1; then
    echo "✅ Docker daemon accessible"
    
    # Test image pull through proxy
    echo "🧪 Testing image pull through proxy..."
    if timeout 60 docker pull hello-world:latest >/dev/null 2>&1; then
        echo "✅ Docker registry accessible through proxy"
        docker rmi hello-world:latest >/dev/null 2>&1 || true
    else
        echo "⚠️  Docker registry test failed"
        echo "   This may indicate proxy or SSL certificate issues"
    fi
else
    echo "❌ Docker daemon not accessible"
    exit 1
fi

echo ""
echo "✅ Colima proxy setup completed!"
echo ""
echo "📋 Next steps:"
echo "1. Add Zscaler certificates to certs/ directory if needed"
echo "2. Use scripts/docker-build-colima.sh for building with proxy support"
echo "3. Test: docker run --rm -it alpine:latest sh"
