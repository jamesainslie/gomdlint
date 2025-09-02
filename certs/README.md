# GEICO Corporate Certificates

This directory contains GEICO corporate CA certificates required for secure communication within the GEICO network environment.

## Certificate Files

Place the following GEICO corporate certificates in this directory:

- `geico-ca.crt` - Primary GEICO Certificate Authority
- `geico-intermediate-ca.crt` - Intermediate Certificate Authority (if used)
- `zscaler-ca.crt` - Zscaler proxy certificates (if applicable)

## Installation

### Automatic Installation

Run the GEICO environment setup script:

```bash
./scripts/geico-env-setup.sh
```

This script will automatically detect your operating system and install the certificates to the appropriate trust store.

### Manual Installation

#### macOS
```bash
sudo security add-trusted-cert -d -r trustRoot -k "/Library/Keychains/System.keychain" certs/geico-ca.crt
```

#### Linux (Ubuntu/Debian)
```bash
sudo cp certs/*.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

#### Windows
1. Double-click each `.crt` file
2. Choose "Install Certificate"
3. Select "Local Machine" 
4. Choose "Place all certificates in the following store"
5. Select "Trusted Root Certification Authorities"

## Environment-Specific Notes

### GEICO IN (Development) Environment
- All certificates should be automatically available
- No manual installation typically required

### GEICO PD (Staging) Environment  
- Certificates are pre-installed on staging infrastructure
- Manual installation may be required for development access

### GEICO UT (Production) Environment
- Certificates are pre-installed on production infrastructure
- Strict certificate validation enforced

### Local Development
- Manual certificate installation required
- Contact GEICO IT for certificate files
- Use `make geico-proxy-test` to verify connectivity

## Docker Integration

For Docker builds, certificates in this directory are automatically copied to the container:

```dockerfile
COPY certs/ /usr/local/share/ca-certificates/
RUN update-ca-certificates
```

## Security Notes

⚠️ **Important Security Guidelines:**

1. **Never commit certificate files to version control**
2. Certificates are already in `.gitignore`
3. Only use official GEICO-issued certificates  
4. Report any certificate-related security issues to GEICO IT
5. Certificates should be obtained through official GEICO channels

## Obtaining Certificates

Contact your GEICO IT team or security team to obtain the required certificates for your development environment.

**Internal Teams:** Check the GEICO internal documentation for certificate distribution procedures.

## Troubleshooting

### Common Issues

1. **"Certificate not trusted" errors**
   - Verify certificate installation with `make geico-env-check`
   - Check certificate validity dates
   - Ensure using official GEICO certificates

2. **Go proxy connectivity issues**
   - Test with `make geico-proxy-test`
   - Verify network connectivity from your environment
   - Check if you're behind Zscaler or other corporate proxy

3. **Docker build failures**
   - Ensure certificates are in `certs/` directory before building
   - Check Docker has access to certificate files
   - Verify certificate format (should be PEM-encoded)

### Getting Help

- **Internal GEICO Teams:** Use internal IT support channels
- **External Contributors:** Certificate setup not required for external contributions
- **CI/CD Issues:** Check Azure DevOps or GitHub Actions logs for certificate-related errors

## Environment Variables

The following environment variables are used for certificate handling:

- `GEICO_ENV` - Detected GEICO environment (in/pd/ut/local)
- `SSL_CERT_FILE` - Custom certificate bundle (if needed)
- `REQUESTS_CA_BUNDLE` - Python requests certificate bundle (if applicable)
