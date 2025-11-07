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
MODE=all                    # all, transfer, deploy, or parallel
MAX_TRANSACTIONS=10000
DELAY_SECONDS=1
VALUE=1
GAS_LIMIT=210000

# Parallel mode settings (only for MODE=parallel)
MIN_BALANCE=100000          # Minimum balance to create wallets (wei)
WALLET_COUNT=1000          # Number of wallets to create
FUNDING_AMOUNT=100         # Amount to fund each wallet (wei)
```

## Modes

- **`all`** (default) - Runs transfers and contract operations in parallel
- **`transfer`** - Sends transactions to 25 random addresses
- **`deploy`** - Deploys auto-generated smart contracts
- **`parallel`** - Maximum TPS mode: Creates 1000 wallets (if balance > 100,000 wei), funds them, and sends transactions from ALL wallets simultaneously with no delays

## Features

- ✅ Auto-generated contracts (no manual bytecode needed)
- ✅ Sends to 25 randomly generated addresses
- ✅ Thread-safe nonce management for parallel operations
- ✅ Automatic contract deployment and interaction
- ✅ **NEW**: Multi-wallet parallel mode - Creates 1000 wallets and sends transactions from all simultaneously

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

## Parallel Mode (Maximum TPS)

The `parallel` mode is designed for maximum transaction throughput:

1. **Checks balance**: If balance > 100,000 wei, creates 1000 new wallets
2. **Funds wallets**: Sends 100 wei to each new wallet in parallel
3. **Sends transactions**: All wallets (original + 1000 new) send transactions simultaneously
4. **No delays**: Maximum throughput with no artificial delays

**Example:**
```bash
MODE=parallel
MAX_TRANSACTIONS=1000
MIN_BALANCE=100000
WALLET_COUNT=1000
FUNDING_AMOUNT=100
```

This will create 1000 wallets and send 1,001,000 total transactions (1 from original + 1000 from each of 1000 wallets).

## Project Structure

```
├── cmd/simulator/          # Main application
├── internal/
│   ├── config/             # Configuration (.env loader)
│   ├── transaction/        # Transaction sending + nonce management
│   ├── contract/           # Contract deployment & interaction
│   └── wallet/             # Wallet generation & management
├── scripts/
│   ├── start-local-node.sh # Start Geth dev node
│   ├── extract-key.go     # Extract private key from keystore
│   └── test.sh            # Verify setup
└── .env.example           # Configuration template
```
