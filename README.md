# Ethereum Transaction Simulator

**Stress testing tool for EVM-compatible blockchains**

This tool is designed to stress test your blockchain and verify that transactions and smart contract deployment work correctly on EVM-compatible chains.

## Purpose

- **Stress Testing**: Test maximum TPS (transactions per second) of your blockchain
- **Functionality Verification**: Verify that transactions and smart contract deployment/interaction work correctly
- **Network Load Testing**: Generate high transaction volume to test network capacity

## Quick Start

### 1. Setup

```bash
# Copy environment file
cp .env.example .env

# Edit .env and set your private key and RPC URL
nano .env
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

### 3. Run Stress Test

```bash
go run cmd/simulator/main.go
```

## Configuration

Edit `.env` file:

```bash
# Required
PRIVATE_KEY=your_private_key
RPC_URL=http://127.0.0.1:8545

# Modes
MODE=parallel          # parallel, all, transfer, or deploy

# Transaction Settings
VALUE=1                 # Amount to send per transaction (wei)
GAS_LIMIT=210000       # Gas limit per transaction
MAX_TRANSACTIONS=10000 # Not used in parallel mode

# Parallel Mode (Maximum Stress Test)
MIN_BALANCE=100000     # Minimum balance to create wallets (wei)
WALLET_COUNT=1000      # Number of wallets to create
FUNDING_AMOUNT=100     # Amount to fund each wallet (wei)
```

## Modes

- **`parallel`** (recommended for stress testing) - Creates 1000 wallets and sends transactions continuously from all wallets until balance runs out. Maximum TPS mode with no delays.
- **`all`** - Runs transfers and contract operations in parallel
- **`transfer`** - Sends transactions to 25 random addresses
- **`deploy`** - Deploys auto-generated smart contracts

## How It Works

### Parallel Mode (Stress Test)

1. **Checks balance**: If balance > 100,000 wei, creates 1000 new wallets
2. **Funds wallets**: Sends 100 wei to each new wallet in parallel
3. **Continuous transactions**: All wallets send transactions simultaneously until balance runs out
4. **No delays**: Maximum throughput - transactions fire as fast as possible

This mode is designed for maximum stress testing. Transactions continue until all wallets are exhausted.

### Contract Testing

The tool automatically:
- Generates simple storage contracts
- Deploys them to the blockchain
- Interacts with deployed contracts
- Verifies contract functionality works

## Requirements

- Go 1.20+
- EVM-compatible RPC endpoint
- Private key with sufficient balance
- (Optional) Geth for local testing

## Troubleshooting

**"PRIVATE_KEY required"**
- Set `PRIVATE_KEY` in `.env` file

**"failed to connect to RPC"**
- Start local node: `./scripts/start-local-node.sh`
- Or update `RPC_URL` in `.env` to point to your blockchain RPC

**"replacement transaction underpriced"**
- Fixed - uses thread-safe nonce management

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
│   └── extract-key.go      # Extract private key from keystore
└── .env.example            # Configuration template
```

## Use Cases

- **Blockchain Development**: Test your EVM-compatible chain before mainnet
- **TPS Benchmarking**: Measure maximum transactions per second
- **Network Stress Testing**: Test network capacity under load
- **Functionality Verification**: Ensure transactions and contracts work correctly
