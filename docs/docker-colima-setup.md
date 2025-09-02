# Docker Setup for Colima with Zscaler Proxy

This guide helps set up Docker builds in Colima environments with Zscaler SSL interception and corporate proxies.

## Common Issues

### 1. 127.0.0.1 Proxy Not Accessible from Containers

**Problem**: Proxy running on `127.0.0.1:9000` is not accessible from inside Colima containers.

**Solution**: Use host IP instead of localhost in container builds.

```bash
# Use the setup script
make setup-colima-proxy

# Or manual Colima restart with proxy
colima stop
colima start --http-proxy=$HTTP_PROXY --https-proxy=$HTTPS_PROXY
```

### 2. Zscaler SSL Certificate Issues

**Problem**: SSL connections fail due to Zscaler certificate interception.

**Solution**: Add Zscaler certificates to the build context.

```bash
# Find Zscaler certificates
find /usr/local/share/ca-certificates/ -name "*zscaler*" -o -name "*Zscaler*"

# Copy to project
cp /usr/local/share/ca-certificates/ZscalerRootCertificate*.crt certs/

# Build with certificates
make docker-build-colima
```

### 3. Docker Registry Timeouts

**Problem**: Docker registry connections timeout through proxy.

**Solutions**:
1. **Configure Docker daemon proxy** (handled by setup script)
2. **Use local registry** or **pre-pulled images**
3. **Increase timeout values**

## Setup Scripts

### Automatic Colima Configuration

```bash
# Setup Colima with proper proxy configuration
make setup-colima-proxy
```

This script will:
- Detect current proxy settings
- Stop and restart Colima with proxy configuration
- Configure Docker daemon proxy settings
- Test connectivity

### Docker Build with Proxy Support

```bash
# Build with automatic proxy detection and certificate handling
make docker-build-colima
```

This script will:
- Detect Colima host IP automatically
- Update 127.0.0.1 proxy URLs to use host IP
- Pass proxy arguments to Docker build
- Handle Zscaler certificate installation
- Test the built image

## Manual Configuration

### 1. Colima Proxy Configuration

```bash
# Stop Colima
colima stop

# Start with proxy configuration
colima start \
    --cpu 4 \
    --memory 8 \
    --http-proxy=$HTTP_PROXY \
    --https-proxy=$HTTPS_PROXY
```

### 2. Docker Daemon Proxy

If the above doesn't work, configure Docker daemon directly:

```bash
# SSH into Colima VM
colima ssh

# Create proxy configuration
sudo mkdir -p /etc/systemd/system/docker.service.d
echo '[Service]
Environment="HTTP_PROXY='$HTTP_PROXY'"
Environment="HTTPS_PROXY='$HTTPS_PROXY'"
Environment="NO_PROXY='$NO_PROXY'"' | sudo tee /etc/systemd/system/docker.service.d/http-proxy.conf

# Restart Docker
sudo systemctl daemon-reload
sudo systemctl restart docker
```

### 3. Zscaler Certificate Installation

```bash
# Find Zscaler certificates on macOS
find ~/Library/Application\ Support/ZScaler/ -name "*.crt" -o -name "*.pem"

# Copy to project
cp path/to/zscaler/cert.crt certs/zscaler-ca.crt

# Verify in build
docker build --build-arg HTTP_PROXY=$HTTP_PROXY .
```

## Environment Variables

Set these in your shell profile (`~/.zshrc`, `~/.bashrc`):

```bash
# Corporate proxy (adjust to your setup)
export HTTP_PROXY=http://127.0.0.1:9000
export HTTPS_PROXY=http://127.0.0.1:9000
export NO_PROXY=localhost,127.0.0.1,*.local

# For Docker builds
export DOCKER_BUILDKIT=1
```

## Troubleshooting

### Test Connectivity

```bash
# Test proxy connectivity from host
curl -v --proxy $HTTP_PROXY https://registry-1.docker.io

# Test from inside Colima
colima ssh -- curl -v https://registry-1.docker.io

# Test Docker info
docker info | grep -i proxy
```

### Common Solutions

1. **Restart Colima with fresh configuration**:
   ```bash
   colima delete
   make setup-colima-proxy
   ```

2. **Check certificate trust**:
   ```bash
   # On macOS
   security find-certificate -a -Z | grep -i zscaler
   
   # In container
   docker run --rm alpine:latest cat /etc/ssl/certs/ca-certificates.crt | grep -i zscaler
   ```

3. **Use alternative base image**:
   ```dockerfile
   # Try Ubuntu instead of Alpine if certificate issues persist
   FROM golang:1.21-bullseye AS builder
   ```

4. **Skip Docker builds temporarily**:
   ```yaml
   # In .goreleaser.yml
   # dockers:
   #   - image_templates: [...]
   ```

## Colima Configuration File

Create `~/.colima/default/colima.yaml`:

```yaml
cpu: 4
memory: 8
disk: 60
runtime: docker
network:
  address: true
  dns: [8.8.8.8, 1.1.1.1]
proxy:
  http_proxy: $HTTP_PROXY
  https_proxy: $HTTPS_PROXY
  no_proxy: $NO_PROXY
```

## Performance Notes

- **CPU/Memory**: Increase for faster builds
- **Network**: DNS resolution can be slow with Zscaler
- **Storage**: Ensure sufficient disk space for images
- **Proxy**: Direct connections may be faster than proxy for some registries

## Getting Help

If you continue to experience issues:
1. Check Colima logs: `colima logs`
2. Check Docker daemon logs: `colima ssh -- sudo journalctl -u docker`
3. Verify network connectivity: `colima ssh -- ping 8.8.8.8`
4. Test proxy from VM: `colima ssh -- curl -v --proxy http://HOST_IP:9000 https://google.com`
