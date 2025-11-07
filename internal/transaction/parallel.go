package transaction

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ParallelSender handles parallel transactions from multiple wallets
type ParallelSender struct {
	client     *ethclient.Client
	chainID    *big.Int
	wallets    []*ParallelWallet
	recipients []common.Address
	config     *ParallelConfig
}

// ParallelWallet represents a wallet for parallel sending
type ParallelWallet struct {
	PrivateKey   *ecdsa.PrivateKey
	Address      common.Address
	NonceManager *NonceManager
}

// ParallelConfig holds configuration for parallel transactions
type ParallelConfig struct {
	Value           *big.Int
	GasLimit        uint64
	Data            []byte
	MaxTransactions int
}

// NewParallelSender creates a new parallel transaction sender
func NewParallelSender(client *ethclient.Client, chainID *big.Int, wallets []*ParallelWallet, recipients []common.Address, config *ParallelConfig) *ParallelSender {
	return &ParallelSender{
		client:     client,
		chainID:    chainID,
		wallets:    wallets,
		recipients: recipients,
		config:     config,
	}
}

// SendParallelTransactions sends transactions from all wallets in parallel
func (ps *ParallelSender) SendParallelTransactions(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(ps.wallets)*ps.config.MaxTransactions)
	
	// Use semaphore to limit concurrent operations (avoid overwhelming the node)
	semaphore := make(chan struct{}, 100)

	for _, wallet := range ps.wallets {
		wg.Add(1)
		go func(w *ParallelWallet) {
			defer wg.Done()
			
			rng := rand.New(rand.NewSource(rand.Int63()))
			
			for i := 0; i < ps.config.MaxTransactions; i++ {
				// Acquire semaphore
				semaphore <- struct{}{}
				
				go func(txNum int) {
					defer func() { <-semaphore }()
					
					// Select random recipient
					randomIndex := rng.Intn(len(ps.recipients))
					recipient := ps.recipients[randomIndex]

					nonce, err := w.NonceManager.GetNextNonce(ctx)
					if err != nil {
						errChan <- fmt.Errorf("wallet %s: failed to get nonce: %w", w.Address.Hex(), err)
						return
					}

					gasPrice, err := ps.client.SuggestGasPrice(ctx)
					if err != nil {
						errChan <- fmt.Errorf("wallet %s: failed to get gas price: %w", w.Address.Hex(), err)
						return
					}

					tx := types.NewTransaction(
						nonce,
						recipient,
						ps.config.Value,
						ps.config.GasLimit,
						gasPrice,
						ps.config.Data,
					)

					signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ps.chainID), w.PrivateKey)
					if err != nil {
						errChan <- fmt.Errorf("wallet %s: failed to sign transaction: %w", w.Address.Hex(), err)
						return
					}

					if err := ps.client.SendTransaction(ctx, signedTx); err != nil {
						errChan <- fmt.Errorf("wallet %s: failed to send transaction: %w", w.Address.Hex(), err)
						return
					}
				}(i)
			}
		}(wallet)
	}

	wg.Wait()
	close(errChan)

	// Collect errors (but don't fail on all errors - some may be expected)
	errorCount := 0
	for err := range errChan {
		if err != nil {
			errorCount++
			// Log first few errors for debugging
			if errorCount <= 5 {
				fmt.Printf("Transaction error: %v\n", err)
			}
		}
	}

	if errorCount > 0 {
		fmt.Printf("Total transaction errors: %d out of %d attempted\n", 
			errorCount, len(ps.wallets)*ps.config.MaxTransactions)
	}

	return nil
}

