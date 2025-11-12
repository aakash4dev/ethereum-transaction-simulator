# Ethereum Transaction Simulator

**Stress testing tool for EVM-compatible blockchains**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go)](https://golang.org/)

This tool is designed to stress test your blockchain and verify that transactions and smart contract deployment work correctly on EVM-compatible chains.

## Table of Contents

- [Purpose](#purpose)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Modes](#modes)
- [How It Works](#how-it-works)
- [Requirements](#requirements)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Purpose

- **Stress Testing**: Test maximum TPS (transactions per second) of your blockchain
- **Functionality Verification**: Verify that transactions and smart contract deployment/interaction work correctly
- **Network Load Testing**: Generate high transaction volume to test network capacity

## Quick Start

### 1. Prerequisites

- Go 1.20 or higher
- EVM-compatible RPC endpoint
- Private key with sufficient balance
- (Optional) Geth for local testing

### 2. Installation

```bash
# Clone the repository
git clone https://github.com/aakash4dev/ethereum-transaction-simulator.git
cd ethereum-transaction-simulator

# Install dependencies
go mod download
```

### 3. Configuration

```bash
# Copy environment file
cp .env.example .env

# Edit .env and set your private key and RPC URL
nano .env  # or use your preferred editor
```

**Required in `.env`:**
```bash
PRIVATE_KEY=your_private_key_here
RPC_URL=http://127.0.0.1:8545
```

### 4. Start Local Node (Optional)

For local testing with Geth:

```bash
./scripts/start-local-node.sh
```

**Get private key from local node:**
```bash
go run scripts/extract-key.go $(find ./local-node-data/keystore -type f | head -1)
```

### 5. Run Stress Test

**Option 1: Direct Go execution**
```bash
go run cmd/simulator/main.go
```

**Option 2: Build and run**
```bash
go build -o simulator ./cmd/simulator
./simulator
```

**Option 3: Using Docker**
```bash
# Build the Docker image
docker build -t ethereum-simulator .

# Run with environment variables
docker run --env-file .env ethereum-simulator

# Or run with docker-compose
docker-compose up
```

## Configuration

Edit `.env` file with your settings:

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
DELAY_SECONDS=1        # Delay between transactions in seconds (not used in parallel mode)
RETRY_DELAY=10         # Delay before retrying failed operations (seconds)

# Parallel Mode (Maximum Stress Test)
MIN_BALANCE=100000     # Minimum balance to create wallets (wei)
WALLET_COUNT=1000      # Number of wallets to create
FUNDING_AMOUNT=100     # Amount to fund each wallet (wei)
MAX_CONCURRENT_REQUESTS=2000  # Maximum concurrent RPC requests
BALANCE_CHECK_INTERVAL=100    # Check balance every N transactions
FUNDING_CONCURRENCY=50        # Concurrent funding operations
```

## Modes

### `parallel` (Recommended for Stress Testing)
Creates 1000 wallets and sends transactions continuously from all wallets until balance runs out. Maximum TPS mode with no delays.

### `all`
Runs transfers and contract operations in parallel.

### `transfer`
Sends transactions to 25 random addresses.

### `deploy`
Deploys auto-generated smart contracts.

## Features

- ✅ **Graceful Shutdown**: Press Ctrl+C to safely stop the simulator
- ✅ **Error Handling**: Comprehensive error reporting and retry logic
- ✅ **Transaction Verification**: Automatic verification of sent transactions
- ✅ **Configurable Rate Limiting**: Control concurrent requests to protect RPC nodes
- ✅ **Balance Caching**: Optimized balance checks to reduce RPC calls
- ✅ **Input Validation**: Validates all configuration before execution
- ✅ **Multiple Modes**: Support for transfers, deployments, interactions, and parallel stress testing

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

### For Direct Execution
- **Go**: 1.20 or higher
- **EVM-compatible RPC endpoint**: Any blockchain with Ethereum-compatible RPC
- **Private key**: With sufficient balance for testing
- **Optional**: Geth for local node testing

### For Docker Execution
- **Docker**: Docker Engine or Docker Desktop installed
- **EVM-compatible RPC endpoint**: Any blockchain with Ethereum-compatible RPC
- **Private key**: With sufficient balance for testing

## Troubleshooting

### "PRIVATE_KEY required"
- Set `PRIVATE_KEY` in `.env` file
- Ensure the key is in hex format (with or without `0x` prefix)

### "failed to connect to RPC"
- Start local node: `./scripts/start-local-node.sh`
- Or update `RPC_URL` in `.env` to point to your blockchain RPC
- Check if the RPC endpoint is accessible

### "replacement transaction underpriced"
- This is fixed - the tool uses thread-safe nonce management
- If you still see this, ensure you're using the latest version

### "insufficient funds"
- Ensure your wallet has sufficient balance
- Check gas price and gas limit settings
- For parallel mode, ensure balance > MIN_BALANCE

## Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) for details on:

- Code of conduct
- Development process
- Code style guidelines
- How to submit pull requests

See [CONTRIBUTING.md](CONTRIBUTING.md) for more information.

## Use Cases

- **Blockchain Development**: Test your EVM-compatible chain before mainnet
- **TPS Benchmarking**: Measure maximum transactions per second
- **Network Stress Testing**: Test network capacity under load
- **Functionality Verification**: Ensure transactions and contracts work correctly

## Project Structure

```
├── cmd/
│   └── simulator/
│       └── main.go         # Main application entry point
├── internal/
│   ├── config/             # Configuration (.env loader & validation)
│   ├── transaction/        # Transaction sending + nonce management
│   │   ├── parallel.go     # Parallel transaction sender
│   │   ├── sender.go       # Sequential transaction sender
│   │   └── nonce.go        # Thread-safe nonce manager
│   ├── contract/           # Contract deployment & interaction
│   │   ├── deployer.go     # Contract deployment logic
│   │   └── generator.go    # Contract bytecode generation
│   └── wallet/             # Wallet generation & management
│       └── manager.go      # Wallet manager for parallel mode
├── scripts/
│   ├── start-local-node.sh # Start Geth dev node
│   ├── extract-key.go      # Extract private key from keystore
│   └── test.sh             # Test script
├── .env.example            # Configuration template
├── Dockerfile              # Docker image definition
├── docker-compose.yml      # Docker Compose configuration
├── CONTRIBUTING.md         # Contribution guidelines
└── LICENSE                 # MIT License
```

## License

Copyright (c) 2024 Ethereum Transaction Simulator Contributors

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Disclaimer

This tool is for testing and development purposes only. Use responsibly and only on test networks or with permission on private networks. The authors are not responsible for any misuse of this software.
