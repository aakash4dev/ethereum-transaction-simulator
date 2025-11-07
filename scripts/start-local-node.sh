#!/bin/bash

# Start a local Ethereum node for testing
# This script starts a Geth dev node with a pre-funded account

set -e

echo "Starting local Ethereum node..."

# Check if geth is installed
if ! command -v geth &> /dev/null; then
    echo "Error: Geth is not installed."
    echo "Install it from: https://geth.ethereum.org/docs/getting-started/installing-geth"
    exit 1
fi

# Create data directory if it doesn't exist
DATA_DIR="./local-node-data"
if [ ! -d "$DATA_DIR" ]; then
    mkdir -p "$DATA_DIR"
fi

# Generate a new account if it doesn't exist
ACCOUNT_FILE="$DATA_DIR/account.txt"
if [ ! -f "$ACCOUNT_FILE" ]; then
    echo "Creating new account..."
    geth account new --datadir "$DATA_DIR" --password <(echo "") > "$ACCOUNT_FILE" 2>&1
    ACCOUNT=$(grep -o "0x[a-fA-F0-9]\{40\}" "$ACCOUNT_FILE" | head -1)
    echo "Created account: $ACCOUNT"
    echo "$ACCOUNT" > "$ACCOUNT_FILE"
else
    ACCOUNT=$(cat "$ACCOUNT_FILE")
    echo "Using existing account: $ACCOUNT"
fi

# Start the dev node
echo "Starting Geth dev node on http://127.0.0.1:8545"
echo "Account with funds: $ACCOUNT"
echo ""
echo "To get the private key for this account, run:"
echo "  ./scripts/get-private-key.sh"
echo "Or:"
echo "  go run scripts/extract-key.go \$(find ./local-node-data/keystore -name '*$(echo $ACCOUNT | tr '[:upper:]' '[:lower:]' | sed 's/0x//')*')"
echo ""
echo "Press Ctrl+C to stop the node"
echo ""

geth \
    --dev \
    --datadir "$DATA_DIR" \
    --http \
    --http.addr "127.0.0.1" \
    --http.port 8545 \
    --http.api "eth,net,web3,personal" \
    --http.corsdomain "*" \
    --allow-insecure-unlock \
    --dev.period 0

