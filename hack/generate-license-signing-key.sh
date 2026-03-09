#!/bin/sh
set -eu

PRIVATE_KEY_PEM=$(openssl genpkey -algorithm ed25519 2>/dev/null)
PUBLIC_KEY_PEM=$(echo "$PRIVATE_KEY_PEM" | openssl pkey -pubout 2>/dev/null)

echo "LICENSE_KEY_PRIVATE_KEY=$(echo "$PRIVATE_KEY_PEM" | base64 -w0 2>/dev/null || echo "$PRIVATE_KEY_PEM" | base64)"
echo "LICENSE_KEY_PUBLIC_KEY=$(echo "$PUBLIC_KEY_PEM" | base64 -w0 2>/dev/null || echo "$PUBLIC_KEY_PEM" | base64)"
