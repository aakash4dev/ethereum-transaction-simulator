# Fix nonce conflicts in parallel transaction operations

## Problem
When running the simulator in "all" mode (parallel transfers and contract deployments), transactions were failing with "replacement transaction underpriced" errors due to nonce conflicts.

## Root Cause
Both transfer and deploy operations were fetching nonces independently from the same account simultaneously, causing them to receive the same nonce value and attempt to send replacement transactions.

## Solution
Implemented a thread-safe `NonceManager` that:
- Uses mutex synchronization to ensure atomic nonce allocation
- Shares a single nonce manager instance between parallel operations
- Provides sequential nonce assignment without conflicts

## Changes Made
- Added `internal/transaction/nonce.go` with `NonceManager` implementation
- Updated `Sender` and `Deployer` to use shared nonce manager in parallel mode
- Fixed contract bytecode hex string length issue
- Simplified README for better user experience

## Testing
- Verified parallel operations work without nonce conflicts
- Tested with local Geth node
- Confirmed sequential nonce assignment

## Labels
bug, enhancement, fixed

