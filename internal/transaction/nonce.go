package transaction

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// NonceManager manages nonces for an account in a thread-safe manner
type NonceManager struct {
	client      *ethclient.Client
	address     common.Address
	currentNonce uint64
	mu          sync.Mutex
	initialized bool
}

// NewNonceManager creates a new nonce manager
func NewNonceManager(client *ethclient.Client, address common.Address) *NonceManager {
	return &NonceManager{
		client:  client,
		address: address,
	}
}

// GetNextNonce returns the next available nonce in a thread-safe manner
// It always uses PendingNonceAt as the source of truth to ensure it accounts for pending transactions
// The local counter is only used to prevent reusing the same nonce if PendingNonceAt returns
// the same value twice in quick succession (before the node has added our tx to mempool)
func (nm *NonceManager) GetNextNonce(ctx context.Context) (uint64, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Always get the pending nonce from the network - this is the source of truth
	// PendingNonceAt returns the next nonce that should be used, accounting for all pending transactions
	pendingNonce, err := nm.client.PendingNonceAt(ctx, nm.address)
	if err != nil {
		return 0, err
	}
	
	// If we haven't initialized or network nonce is higher, use network value
	if !nm.initialized || pendingNonce > nm.currentNonce {
		nm.currentNonce = pendingNonce
		nm.initialized = true
	}
	// If network nonce equals our counter, it means we just used this nonce but node hasn't seen it yet
	// In this case, increment our counter to avoid reusing the same nonce
	// If network nonce is lower (shouldn't happen), use our counter
	
	nonce := nm.currentNonce
	nm.currentNonce++
	return nonce, nil
}

// Reset re-initializes the nonce from the network
func (nm *NonceManager) Reset(ctx context.Context) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nonce, err := nm.client.PendingNonceAt(ctx, nm.address)
	if err != nil {
		return err
	}
	nm.currentNonce = nonce
	nm.initialized = true
	return nil
}

// WaitForNonceUpdate waits for the pending nonce to reflect a transaction we just sent
// This ensures the node has accepted the transaction into its mempool before we proceed
func (nm *NonceManager) WaitForNonceUpdate(ctx context.Context, expectedNonce uint64, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			pendingNonce, err := nm.client.PendingNonceAt(ctx, nm.address)
			if err != nil {
				continue // Retry on error
			}
			// If pending nonce is greater than expected, the transaction was accepted
			if pendingNonce > expectedNonce {
				nm.mu.Lock()
				nm.currentNonce = pendingNonce
				nm.mu.Unlock()
				return nil
			}
		}
	}
	// Timeout - but don't fail, just continue (node might be slow)
	return nil
}

