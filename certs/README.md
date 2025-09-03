# Corporate Certificate Configuration

This directory is for corporate SSL certificates needed to access private package repositories and registries through corporate proxies (like Zscaler).

## Certificate Files

Place your corporate certificates in this directory:

```
certs/
├── corporate-ca.crt          # Corporate root CA certificate
├── intermediate-ca.crt       # Intermediate CA certificate (if needed)
├── zscaler-root.crt         # Zscaler root certificate (if using Zscaler)
└── README.md                # This file
```

## Installation

### Manual Installation

#### macOS
```bash
sudo security add-trusted-cert -d -r trustRoot -k "/Library/Keychains/System.keychain" certs/corporate-ca.crt
```

#### Linux (Ubuntu/Debian)
```bash
sudo cp certs/*.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

#### Windows
```powershell
# Import certificate to trusted root store
Import-Certificate -FilePath "certs\corporate-ca.crt" -CertStoreLocation "Cert:\LocalMachine\Root"
```

## Docker Integration

The Dockerfile automatically handles certificate installation:

```dockerfile
# Certificates are copied and installed during Docker build
COPY certs /tmp/certs/
RUN find /tmp/certs -name "*.crt" -exec cp {} /usr/local/share/ca-certificates/ \; || true
RUN update-ca-certificates
```

## Security Notes

1. **Never commit certificate files to version control**
2. Certificates are already in `.gitignore`
3. Only use official corporate-issued certificates  
4. Report any certificate-related security issues to your IT security team
5. Certificates should be obtained through official corporate channels

## Obtaining Certificates

Contact your corporate IT team or security team to obtain the required certificates for your development environment.

**Corporate Teams:** Check your internal documentation for certificate distribution procedures.

## Troubleshooting

### Common Issues

1. **"Certificate not trusted" errors**
   - Verify certificate installation with system certificate tools
   - Check certificate validity dates
   - Ensure using official corporate certificates

2. **Go proxy connectivity issues**
   - Test with `make proxy-test`
   - Verify network connectivity from your environment
   - Check if you're behind Zscaler or other corporate proxy

3. **Docker build failures**
   - Ensure certificates are in `certs/` directory before building
   - Check Docker build logs for certificate installation messages

### Getting Help

- **Corporate Teams:** Use internal IT support channels
- **External Contributors:** Certificate setup not required for external contributions
- **CI/CD Issues:** Check GitHub Actions logs for certificate-related errors

## Environment Variables

The following environment variables are used for certificate handling:

- `BUILD_ENV` - Detected build environment (ci/local)
- `SSL_CERT_FILE` - Custom certificate bundle (if needed)
- `REQUESTS_CA_BUNDLE` - Python requests certificate bundle (if applicable)