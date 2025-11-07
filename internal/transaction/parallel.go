package transaction

import (
	"context"
	"crypto/ecdsa"
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

// SendParallelTransactions sends transactions continuously from all wallets until balance runs out
func (ps *ParallelSender) SendParallelTransactions(ctx context.Context) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 2000)

	// Launch continuous transaction sending from each wallet
	for _, wallet := range ps.wallets {
		wg.Add(1)
		go func(w *ParallelWallet) {
			defer wg.Done()
			
			rng := rand.New(rand.NewSource(rand.Int63()))
			balanceCheckCounter := 0
			
			// Continuous loop - send transactions until balance runs out
			for {
				// Check balance every 100 transactions to avoid slowing down
				balanceCheckCounter++
				if balanceCheckCounter%100 == 0 {
					balance, err := ps.client.BalanceAt(ctx, w.Address, nil)
					if err != nil {
						return
					}

					gasPrice, err := ps.client.SuggestGasPrice(ctx)
					if err != nil {
						return
					}

					minRequired := new(big.Int).Mul(gasPrice, big.NewInt(int64(ps.config.GasLimit)))
					minRequired.Add(minRequired, ps.config.Value)

					if balance.Cmp(minRequired) < 0 {
						return // Wallet out of balance
					}
				}

				// Acquire semaphore (non-blocking)
				select {
				case semaphore <- struct{}{}:
					// Send transaction immediately
					go func() {
						defer func() { <-semaphore }()
						
						recipient := ps.recipients[rng.Intn(len(ps.recipients))]

						nonce, _ := w.NonceManager.GetNextNonce(ctx)
						gasPrice, _ := ps.client.SuggestGasPrice(ctx)

						tx := types.NewTransaction(
							nonce,
							recipient,
							ps.config.Value,
							ps.config.GasLimit,
							gasPrice,
							ps.config.Data,
						)

						signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(ps.chainID), w.PrivateKey)
						ps.client.SendTransaction(ctx, signedTx)
					}()
				default:
					// Semaphore full, continue immediately without blocking
				}
			}
		}(wallet)
	}

	wg.Wait()
	return nil
}

