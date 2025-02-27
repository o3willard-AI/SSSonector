#!/bin/bash

# generate-certs.sh
# Generates certificates for SSSonector
set -euo pipefail

# Default settings
CERT_DIR="certs"
SERVER_CN="${SERVER_CN:-localhost}"
CLIENT_CN="${CLIENT_CN:-sssonector-client}"
DAYS=365
KEY_SIZE=2048
COUNTRY="EU"
STATE="Europe"
LOCALITY="European Union"
ORGANIZATION="SSSonector"
CA_CN="SSSonector CA"

# Create OpenSSL config files
create_ca_config() {
    cat > "${CERT_DIR}/ca.cnf" << EOF
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_ca
prompt = no

[req_distinguished_name]
C = ${COUNTRY}
ST = ${STATE}
L = ${LOCALITY}
O = ${ORGANIZATION}
CN = ${CA_CN}

[v3_ca]
basicConstraints = critical,CA:TRUE
keyUsage = critical,keyCertSign,cRLSign
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer
EOF
}

create_server_config() {
    cat > "${CERT_DIR}/server.cnf" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = ${COUNTRY}
ST = ${STATE}
L = ${LOCALITY}
O = ${ORGANIZATION}
CN = ${SERVER_CN}

[v3_req]
basicConstraints = critical,CA:FALSE
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${SERVER_CN}
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF
}

create_client_config() {
    cat > "${CERT_DIR}/client.cnf" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = ${COUNTRY}
ST = ${STATE}
L = ${LOCALITY}
O = ${ORGANIZATION}
CN = ${CLIENT_CN}

[v3_req]
basicConstraints = critical,CA:FALSE
keyUsage = critical,digitalSignature,keyEncipherment
extendedKeyUsage = clientAuth
subjectAltName = DNS:${CLIENT_CN}
EOF
}

# Generate CA certificate
generate_ca() {
    echo "Generating CA certificate..."
    openssl genrsa -out "${CERT_DIR}/ca.key" ${KEY_SIZE}
    chmod 600 "${CERT_DIR}/ca.key"
    openssl req -new -x509 -days ${DAYS} -key "${CERT_DIR}/ca.key" -out "${CERT_DIR}/ca.crt" -config "${CERT_DIR}/ca.cnf"
    chmod 644 "${CERT_DIR}/ca.crt"
}

# Generate server certificate
generate_server_cert() {
    echo "Generating server certificate..."
    openssl genrsa -out "${CERT_DIR}/server.key" ${KEY_SIZE}
    chmod 600 "${CERT_DIR}/server.key"
    openssl req -new -key "${CERT_DIR}/server.key" -out "${CERT_DIR}/server.csr" -config "${CERT_DIR}/server.cnf"
    openssl x509 -req -days ${DAYS} -in "${CERT_DIR}/server.csr" -CA "${CERT_DIR}/ca.crt" -CAkey "${CERT_DIR}/ca.key" \
        -CAcreateserial -out "${CERT_DIR}/server.crt" -extfile "${CERT_DIR}/server.cnf" -extensions v3_req
    chmod 644 "${CERT_DIR}/server.crt"
    rm -f "${CERT_DIR}/server.csr"
}

# Generate client certificate
generate_client_cert() {
    echo "Generating client certificate..."
    openssl genrsa -out "${CERT_DIR}/client.key" ${KEY_SIZE}
    chmod 600 "${CERT_DIR}/client.key"
    openssl req -new -key "${CERT_DIR}/client.key" -out "${CERT_DIR}/client.csr" -config "${CERT_DIR}/client.cnf"
    openssl x509 -req -days ${DAYS} -in "${CERT_DIR}/client.csr" -CA "${CERT_DIR}/ca.crt" -CAkey "${CERT_DIR}/ca.key" \
        -CAcreateserial -out "${CERT_DIR}/client.crt" -extfile "${CERT_DIR}/client.cnf" -extensions v3_req
    chmod 644 "${CERT_DIR}/client.crt"
    rm -f "${CERT_DIR}/client.csr"
}

# Verify certificates
verify_certs() {
    echo "Verifying certificate chain..."
    echo "==============================="
    echo
    echo "Server certificate:"
    openssl verify -CAfile "${CERT_DIR}/ca.crt" "${CERT_DIR}/server.crt"
    echo
    echo "Client certificate:"
    openssl verify -CAfile "${CERT_DIR}/ca.crt" "${CERT_DIR}/client.crt"
}

# Display certificate details
show_cert_details() {
    echo
    echo "Certificate details:"
    echo "==================="
    echo
    echo "CA Certificate:"
    openssl x509 -in "${CERT_DIR}/ca.crt" -noout -text | grep -E "Subject:|Issuer:|Validity|Key Usage|Basic Constraints"
    echo
    echo "Server Certificate:"
    openssl x509 -in "${CERT_DIR}/server.crt" -noout -text | grep -E "Subject:|Issuer:|Validity|Key Usage|Extended Key Usage|Subject Alternative Name"
    echo
    echo "Client Certificate:"
    openssl x509 -in "${CERT_DIR}/client.crt" -noout -text | grep -E "Subject:|Issuer:|Validity|Key Usage|Extended Key Usage"
}

# List generated files
list_files() {
    echo
    echo "Generated files:"
    ls -l "${CERT_DIR}"/*.{key,crt}
}

# Main function
main() {
    # Create certificates directory
    mkdir -p "${CERT_DIR}"

    # Create OpenSSL config files
    create_ca_config
    create_server_config
    create_client_config

    # Generate certificates
    generate_ca
    generate_server_cert
    generate_client_cert

    # Verify and display results
    verify_certs
    show_cert_details
    list_files

    # Clean up config files
    rm -f "${CERT_DIR}"/*.cnf
    rm -f "${CERT_DIR}"/*.srl

    echo
    echo "Certificate generation complete!"
}

# Run main function
main "$@"
