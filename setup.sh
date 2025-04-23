#!/bin/bash

# Exit on error
set -e

# Configuration
# No CA needed
SERVER_SUBJ="/C=US/ST=California/L=SanFrancisco/O=MyOrg/OU=Server/CN=localhost"
CLIENT_SUBJ="/C=US/ST=California/L=SanFrancisco/O=MyOrg/OU=Client/CN=my_secure_client"
CERT_DIR="certs"
KNOWN_CLIENTS_FILE="$CERT_DIR/knownClients.txt"
SERVER_EXT_FILE="$CERT_DIR/server_ext.cnf"

# Clean previous certs
echo "Cleaning up previous certificates..."
rm -rf "$CERT_DIR"
mkdir "$CERT_DIR"

# --- No CA Generation --- 

echo "Generating Self-Signed Server Certificate..."
# Create OpenSSL config file including SAN extension
cat > "$SERVER_EXT_FILE" <<-EOF
[ req ]
distinguished_name = req_distinguished_name
x509_extensions = v3_ca # Using v3_ca section for extensions
req_extensions = v3_req
prompt = no

[ req_distinguished_name ]
C = US
ST = California
L = SanFrancisco
O = MyOrg
OU = Server
CN = localhost

[ v3_ca ]
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints = critical,CA:false
keyUsage = critical, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[ v3_req ] # Section for req_extensions
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = localhost
IP.1 = 127.0.0.1
EOF

# Generate self-signed server certificate directly using the key and subject from the config file
openssl req -x509 -newkey rsa:2048 -nodes -keyout "$CERT_DIR/server.key" \
    -out "$CERT_DIR/server.crt" -days 365 -config "$SERVER_EXT_FILE" -extensions v3_ca # Specify extensions section

echo "Generating Self-Signed Client Certificate..."
# Generate client private key and self-signed certificate directly
# Client cert doesn't usually need SANs, so direct command is fine
openssl req -x509 -newkey rsa:2048 -nodes -keyout "$CERT_DIR/client.key" \
    -out "$CERT_DIR/client.crt" -subj "$CLIENT_SUBJ" -days 365

echo "Generating knownClients.txt..."
# Extract client certificate SHA-256 fingerprint (more robustly)
CLIENT_FINGERPRINT=$(openssl x509 -noout -fingerprint -sha256 -in "$CERT_DIR/client.crt" | cut -d "=" -f 2)
# Use the CN defined in the configuration directly
CLIENT_CN_FROM_CONFIG="my_secure_client"

# Create knownClients.txt
echo "$CLIENT_CN_FROM_CONFIG $CLIENT_FINGERPRINT" > "$KNOWN_CLIENTS_FILE"

# Clean up temporary files
rm "$SERVER_EXT_FILE"
# rm "$CERT_DIR"/*.csr # No CSRs generated in this version

echo "Setup complete. Self-signed certificates and knownClients.txt are in the '$CERT_DIR' directory."
echo "Known Client Entry:"
cat "$KNOWN_CLIENTS_FILE"

# Make keys readable only by owner (best practice)
chmod 400 "$CERT_DIR"/*.key 