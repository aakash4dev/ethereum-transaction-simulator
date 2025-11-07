# Ethereum Transaction Simulator

A Go tool for testing TPS (transactions per second) on EVM-compatible blockchains. Automatically generates contracts, deploys them, and sends transactions to random addresses in parallel.

## Quick Start

### 1. Setup

```bash
# Copy environment file
cp .env.example .env

# Edit .env and set your private key
nano .env  # or use your preferred editor
```

Required in `.env`:
```bash
PRIVATE_KEY=your_private_key_here
RPC_URL=http://127.0.0.1:8545
```

### 2. Start Local Node (Optional)

```bash
./scripts/start-local-node.sh
```

**Get private key from local node:**
```bash
go run scripts/extract-key.go $(find ./local-node-data/keystore -type f | head -1)
```

Copy the private key to your `.env` file.

### 3. Run

```bash
go run cmd/simulator/main.go
```

## Configuration

Edit `.env` file:

```bash
# Required
PRIVATE_KEY=your_private_key

# Optional (defaults shown)
RPC_URL=http://127.0.0.1:8545
MODE=all                    # all, transfer, or deploy
MAX_TRANSACTIONS=10000
DELAY_SECONDS=1
VALUE=1
GAS_LIMIT=210000
```

## Modes

- **`all`** (default) - Runs transfers and contract operations in parallel
- **`transfer`** - Sends transactions to 25 random addresses
- **`deploy`** - Deploys auto-generated smart contracts

## Features

- ✅ Auto-generated contracts (no manual bytecode needed)
- ✅ Sends to 25 randomly generated addresses
- ✅ Thread-safe nonce management for parallel operations
- ✅ Automatic contract deployment and interaction

## Requirements

- Go 1.20+
- EVM-compatible RPC endpoint
- Private key with balance
- (Optional) Geth for local testing

## Troubleshooting

**"PRIVATE_KEY required"**
- Set `PRIVATE_KEY` in `.env` file

**"failed to connect to RPC"**
- Start local node: `./scripts/start-local-node.sh`
- Or update `RPC_URL` in `.env`

**"replacement transaction underpriced"**
- Fixed in v0.0.1 - uses thread-safe nonce management

## Project Structure

```
├── cmd/simulator/          # Main application
├── internal/
│   ├── config/             # Configuration (.env loader)
│   ├── transaction/        # Transaction sending + nonce management
│   └── contract/           # Contract deployment & interaction
├── scripts/
│   ├── start-local-node.sh # Start Geth dev node
│   ├── extract-key.go     # Extract private key from keystore
│   └── test.sh            # Verify setup
└── .env.example           # Configuration template
```
