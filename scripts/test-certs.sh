#!/bin/bash
set -e

# Default values
CERT_DIR="certs"
DAYS=90
COMMON_NAME="SSSonector.local"
IP_ADDRESSES="127.0.0.1,::1"
DNS_NAMES="localhost"

# Help message
show_help() {
    echo "Usage: $0 [options]"
    echo
    echo "Test SSSonector certificate management"
    echo
    echo "Options:"
    echo "  -d, --dir         Certificate directory (default: certs)"
    echo "  --days           Validity period in days (default: 90)"
    echo "  --cn             Common Name (default: SSSonector.local)"
    echo "  --ips            Comma-separated IP addresses (default: 127.0.0.1,::1)"
    echo "  --dns            Comma-separated DNS names (default: localhost)"
    echo "  --help           Show this help message"
    echo
    echo "Example:"
    echo "  $0 --dir /etc/SSSonector/certs --days 365 --cn tunnel.example.com"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--dir)
            CERT_DIR="$2"
            shift 2
            ;;
        --days)
            DAYS="$2"
            shift 2
            ;;
        --cn)
            COMMON_NAME="$2"
            shift 2
            ;;
        --ips)
            IP_ADDRESSES="$2"
            shift 2
            ;;
        --dns)
            DNS_NAMES="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Check if openssl is installed
if ! command -v openssl &> /dev/null; then
    echo "Error: openssl is not installed"
    echo "Ubuntu/Debian: sudo apt-get install openssl"
    echo "RHEL/CentOS: sudo yum install openssl"
    echo "macOS: brew install openssl"
    exit 1
fi

# Create certificate directory
mkdir -p "$CERT_DIR"

# Create test configuration
echo "Testing certificate management"
echo "Directory: $CERT_DIR"
echo "Common Name: $COMMON_NAME"
echo "Validity: $DAYS days"
echo "IP Addresses: $IP_ADDRESSES"
echo "DNS Names: $DNS_NAMES"
echo

# Create test configuration file
cat > "$CERT_DIR/test-config.yaml" << EOF
tls:
  cert_file: "$CERT_DIR/test.crt"
  key_file: "$CERT_DIR/test.key"
  auto_generate: true
  validity_days: $DAYS
  common_name: "$COMMON_NAME"
  ip_addresses:
$(echo "$IP_ADDRESSES" | tr ',' '\n' | sed 's/^/    - /')
  dns_names:
$(echo "$DNS_NAMES" | tr ',' '\n' | sed 's/^/    - /')
EOF

echo "Created test configuration:"
cat "$CERT_DIR/test-config.yaml"
echo

# Generate certificates
echo "Generating test certificates..."
go run cmd/tunnel/main.go --config "$CERT_DIR/test-config.yaml" --test-certs

# Verify certificates
echo
echo "Verifying certificate..."
openssl x509 -in "$CERT_DIR/test.crt" -text -noout | grep -A2 "Validity"
openssl x509 -in "$CERT_DIR/test.crt" -text -noout | grep -A2 "Subject:"
openssl x509 -in "$CERT_DIR/test.crt" -text -noout | grep -A2 "X509v3 Subject Alternative Name:"

# Test certificate chain
echo
echo "Testing certificate chain..."
openssl verify -CAfile "$CERT_DIR/test.crt" "$CERT_DIR/test.crt"

# Test private key match
echo
echo "Testing private key match..."
CERT_MODULUS=$(openssl x509 -in "$CERT_DIR/test.crt" -modulus -noout)
KEY_MODULUS=$(openssl ec -in "$CERT_DIR/test.key" -modulus -noout)
if [ "$CERT_MODULUS" = "$KEY_MODULUS" ]; then
    echo "Certificate and private key match"
else
    echo "ERROR: Certificate and private key do not match"
    exit 1
fi

# Test certificate expiry
echo
echo "Testing certificate expiry..."
EXPIRY=$(openssl x509 -in "$CERT_DIR/test.crt" -enddate -noout | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$EXPIRY" +%s)
NOW_EPOCH=$(date +%s)
DAYS_LEFT=$(( ($EXPIRY_EPOCH - $NOW_EPOCH) / 86400 ))
echo "Certificate expires in $DAYS_LEFT days"

if [ $DAYS_LEFT -lt 7 ]; then
    echo "WARNING: Certificate will expire soon"
elif [ $DAYS_LEFT -lt 0 ]; then
    echo "ERROR: Certificate has expired"
    exit 1
fi

# Test TLS connection
echo
echo "Testing TLS connection..."
echo "Q" | openssl s_client -connect localhost:5000 -cert "$CERT_DIR/test.crt" -key "$CERT_DIR/test.key" 2>/dev/null

echo
echo "Certificate tests completed successfully"
