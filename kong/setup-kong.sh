#!/bin/bash
set -e

# Navigate to repo root (script lives in kong/)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Fetching Keycloak JWT Public Key..."

# Wait for Keycloak
until curl -s http://localhost:8080/realms/prod-maintenance/.well-known/openid-configuration > /dev/null; do
    echo "Waiting for Keycloak to be ready at localhost:8080..."
    sleep 2
done

# Fetch JWKS
JWKS=$(curl -s http://localhost:8080/realms/prod-maintenance/protocol/openid-connect/certs)

# Extract the x5c certificate and convert to RSA public key
CERT_B64=$(echo "$JWKS" | python3 -c "import sys, json; print(json.load(sys.stdin)['keys'][0]['x5c'][0])")

if [ -z "$CERT_B64" ]; then
    echo "Failed to extract certificate from Keycloak."
    exit 1
fi

# Convert x5c certificate → PEM certificate → extract RSA public key
PUBKEY=$(echo "-----BEGIN CERTIFICATE-----"$'\n'"$CERT_B64"$'\n'"-----END CERTIFICATE-----" | openssl x509 -pubkey -noout 2>/dev/null | grep -v "BEGIN\|END" | tr -d '\n')

if [ -z "$PUBKEY" ]; then
    echo "Failed to extract RSA public key from certificate."
    exit 1
fi

echo "Successfully extracted RSA Public Key."

# Format the key into proper PEM (64 chars per line, indented for YAML)
PEM_KEY=$(echo "$PUBKEY" | fold -w 64 | sed 's/^/      /')

# Create a new kong config with the key
cat > "$SCRIPT_DIR/kong.yml" <<ENDOFCONFIG
_format_version: "3.0"

upstreams:
  - name: event-processing-upstream
    targets:
      - target: event-processing-service:8001
        weight: 100
    healthchecks:
      active:
        type: http
        http_path: /health
        timeout: 1
        concurrency: 10
        healthy:
          interval: 5
          successes: 2
        unhealthy:
          interval: 5
          tcp_failures: 2
          http_failures: 2
          timeouts: 2

  - name: ingestion-upstream
    targets:
      - target: ingestion-service:8003
        weight: 100
    healthchecks:
      active:
        type: http
        http_path: /health
        timeout: 1
        concurrency: 10
        healthy:
          interval: 5
          successes: 2
        unhealthy:
          interval: 5
          tcp_failures: 2
          http_failures: 2
          timeouts: 2

services:
  - name: event-processing
    host: event-processing-upstream
    path: /
    routes:
      - name: events-route
        paths:
          - /api/events
        strip_path: true

  - name: ingestion
    host: ingestion-upstream
    path: /
    routes:
      - name: ingestion-route
        paths:
          - /api/ingest
        strip_path: true

plugins:
  - name: cors
    config:
      origins:
        - "*"
      methods:
        - GET
        - POST
        - PUT
        - DELETE
        - OPTIONS
      headers:
        - Accept
        - Accept-Version
        - Content-Length
        - Content-MD5
        - Content-Type
        - Date
        - Authorization
      exposed_headers:
        - Authorization
      credentials: true
      max_age: 3600

  - name: rate-limiting
    config:
      minute: 60
      policy: local

  - name: request-size-limiting
    config:
      allowed_payload_size: 10

  - name: jwt
    config:
      claims_to_verify:
        - exp

consumers:
  - username: keycloak-consumer

jwt_secrets:
  - consumer: keycloak-consumer
    key: "http://localhost:8080/realms/prod-maintenance"
    algorithm: RS256
    rsa_public_key: |
      -----BEGIN PUBLIC KEY-----
${PEM_KEY}
      -----END PUBLIC KEY-----
ENDOFCONFIG

echo "Injected Keycloak public key into $SCRIPT_DIR/kong.yml."

echo "Restarting Kong..."
cd "$REPO_ROOT" && docker-compose rm -f -s kong && docker-compose up -d kong

echo "Done! Kong is now configured with the correct JWT public key."
