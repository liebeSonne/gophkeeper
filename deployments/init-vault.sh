#!/bin/bash
set -euo pipefail

VAULT_ADDR="${VAULT_ADDR:-http://127.0.0.1:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-dev-token}"
KEY_PATH="${KEY_PATH:-secret/gophkeeper/encryption}"

export VAULT_ADDR
export VAULT_TOKEN

echo "Initializing Vault at ${VAULT_ADDR}..."

if ! vault status >/dev/null 2>&1; then
    echo "Error: Vault is not running. Start Vault first: docker compose up -d vault"
    exit 1
fi

if ! vault secrets list | grep -q "^secret/"; then
    echo "Enabling KV v2 at 'secret/'..."
    vault secrets enable -version=2 kv -path=secret
fi

echo "Generating 32-byte encryption key..."
ENCRYPTION_KEY=$(openssl rand -base64 32)

echo "Writing key to Vault at ${KEY_PATH}..."
vault kv put "${KEY_PATH}" key="${ENCRYPTION_KEY}"

echo "Done! Encryption key has been written to Vault."
echo "To use it, set:"
echo "  ENABLE_VAULT=true"
echo "  VAULT_ADDRESS=${VAULT_ADDR}"
echo "  VAULT_TOKEN=${VAULT_TOKEN}"
echo "  VAULT_KEY_PATH=${KEY_PATH}"
