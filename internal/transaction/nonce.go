package transaction

import (
	"context"
	"sync"

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
func (nm *NonceManager) GetNextNonce(ctx context.Context) (uint64, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.initialized {
		nonce, err := nm.client.PendingNonceAt(ctx, nm.address)
		if err != nil {
			return 0, err
		}
		nm.currentNonce = nonce
		nm.initialized = true
	}

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

