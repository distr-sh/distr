#!/bin/sh
set -eu

PRIVATE_KEY_PEM=$(openssl genpkey -algorithm ed25519 2>/dev/null)
PUBLIC_KEY_PEM=$(echo "$PRIVATE_KEY_PEM" | openssl pkey -pubout 2>/dev/null)

echo "LICENSE_KEY_PRIVATE_KEY='$PRIVATE_KEY_PEM'"
echo ""
echo "# Public key (distribute to customers for token verification):"
echo "$PUBLIC_KEY_PEM"
