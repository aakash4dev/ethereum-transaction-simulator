#!/bin/bash

# Test script to verify the simulator is working

set -e

echo "=== Ethereum Transaction Simulator Test ==="
echo ""

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "Error: .env file not found!"
    echo "Please copy .env.example to .env and configure it:"
    echo "  cp .env.example .env"
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

# Build the project
echo "Building project..."
go build -o simulator cmd/simulator/main.go

if [ $? -ne 0 ]; then
    echo "Error: Build failed"
    exit 1
fi

echo "Build successful!"
echo ""

# Check if RPC is accessible
RPC_URL=$(grep "^RPC_URL=" .env | cut -d '=' -f2- | tr -d '"' | tr -d "'")
if [ -z "$RPC_URL" ]; then
    RPC_URL="http://127.0.0.1:8545"
fi

echo "Testing RPC connection to $RPC_URL..."

# Try to connect to RPC (requires curl)
if command -v curl &> /dev/null; then
    RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        "$RPC_URL" 2>/dev/null || echo "")
    
    if echo "$RESPONSE" | grep -q "result"; then
        echo "✓ RPC connection successful"
    else
        echo "⚠ Warning: Could not connect to RPC. Make sure your node is running."
        echo "  Start local node: ./scripts/start-local-node.sh"
    fi
else
    echo "⚠ curl not found, skipping RPC test"
fi

echo ""
echo "=== Configuration Check ==="
PRIVATE_KEY=$(grep "^PRIVATE_KEY=" .env | cut -d '=' -f2- | tr -d '"' | tr -d "'")
MODE=$(grep "^MODE=" .env | cut -d '=' -f2- | tr -d '"' | tr -d "'" || echo "transfer")

if [ -z "$PRIVATE_KEY" ] || [ "$PRIVATE_KEY" = "your_private_key_here" ]; then
    echo "✗ PRIVATE_KEY not configured in .env"
    exit 1
else
    echo "✓ PRIVATE_KEY configured"
fi

echo "✓ MODE: $MODE"

if [ "$MODE" = "deploy" ]; then
    BYTECODE=$(grep "^CONTRACT_BYTECODE=" .env | cut -d '=' -f2- | tr -d '"' | tr -d "'")
    if [ -z "$BYTECODE" ]; then
        echo "✗ CONTRACT_BYTECODE required for deploy mode"
        exit 1
    else
        echo "✓ CONTRACT_BYTECODE configured"
    fi
elif [ "$MODE" = "interact" ]; then
    ADDRESS=$(grep "^CONTRACT_ADDRESS=" .env | cut -d '=' -f2- | tr -d '"' | tr -d "'")
    FUNCTION_DATA=$(grep "^FUNCTION_DATA=" .env | cut -d '=' -f2- | tr -d '"' | tr -d "'")
    if [ -z "$ADDRESS" ] || [ -z "$FUNCTION_DATA" ]; then
        echo "✗ CONTRACT_ADDRESS and FUNCTION_DATA required for interact mode"
        exit 1
    else
        echo "✓ CONTRACT_ADDRESS and FUNCTION_DATA configured"
    fi
fi

echo ""
echo "=== All checks passed! ==="
echo ""
echo "To run the simulator:"
echo "  ./simulator"
echo ""
echo "Or:"
echo "  go run cmd/simulator/main.go"

