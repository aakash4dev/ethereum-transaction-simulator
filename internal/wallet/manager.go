package wallet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/aakash4dev/ethereum-transaction-simulator/internal/transaction"
)

// Wallet represents a wallet with its private key and nonce manager
type Wallet struct {
	PrivateKey  *ecdsa.PrivateKey
	Address     common.Address
	NonceManager *transaction.NonceManager
	Client      *ethclient.Client
}

// Manager manages multiple wallets for parallel transactions
type Manager struct {
	client       *ethclient.Client
	chainID      *big.Int
	fundingAmount *big.Int
}

// NewManager creates a new wallet manager
func NewManager(client *ethclient.Client, chainID *big.Int, fundingAmount *big.Int) *Manager {
	return &Manager{
		client:       client,
		chainID:      chainID,
		fundingAmount: fundingAmount,
	}
}

// GenerateWallets generates n new wallets
func (m *Manager) GenerateWallets(n int) []*Wallet {
	wallets := make([]*Wallet, n)
	for i := 0; i < n; i++ {
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			// Continue with next wallet if generation fails
			continue
		}
		address := crypto.PubkeyToAddress(privateKey.PublicKey)
		nonceManager := transaction.NewNonceManager(m.client, address)

		wallets[i] = &Wallet{
			PrivateKey:   privateKey,
			Address:      address,
			NonceManager: nonceManager,
			Client:       m.client,
		}
	}
	return wallets
}


// FundWallets funds all wallets from the funding wallet in parallel
func (m *Manager) FundWallets(ctx context.Context, fundingWallet *Wallet, wallets []*Wallet) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(wallets))
	semaphore := make(chan struct{}, 50) // Limit concurrent operations

	for _, wallet := range wallets {
		wg.Add(1)
		go func(targetWallet *Wallet) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			nonce, err := fundingWallet.NonceManager.GetNextNonce(ctx)
			if err != nil {
				errChan <- fmt.Errorf("failed to get nonce for funding: %w", err)
				return
			}

			gasPrice, err := m.client.SuggestGasPrice(ctx)
			if err != nil {
				errChan <- fmt.Errorf("failed to get gas price: %w", err)
				return
			}

			tx := types.NewTransaction(
				nonce,
				targetWallet.Address,
				m.fundingAmount,
				21000, // Standard transfer gas limit
				gasPrice,
				nil,
			)

			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(m.chainID), fundingWallet.PrivateKey)
			if err != nil {
				errChan <- fmt.Errorf("failed to sign funding transaction: %w", err)
				return
			}

			if err := m.client.SendTransaction(ctx, signedTx); err != nil {
				errChan <- fmt.Errorf("failed to send funding transaction to %s: %w", targetWallet.Address.Hex(), err)
				return
			}
		}(wallet)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("funding errors: %d wallets failed", len(errors))
	}

	return nil
}

// CheckBalance checks if balance is sufficient
func (m *Manager) CheckBalance(ctx context.Context, address common.Address, minBalance *big.Int) (bool, *big.Int, error) {
	balance, err := m.client.BalanceAt(ctx, address, nil)
	if err != nil {
		return false, nil, err
	}
	return balance.Cmp(minBalance) > 0, balance, nil
}

