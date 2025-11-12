package transaction

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

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
	// Metrics
	totalSent      int64
	totalFailed    int64
	totalSucceeded int64
	errors         []error
	mu             sync.Mutex
}

// ParallelWallet represents a wallet for parallel sending
type ParallelWallet struct {
	PrivateKey   *ecdsa.PrivateKey
	Address      common.Address
	NonceManager *NonceManager
	// Cached balance to reduce RPC calls
	lastBalance     *big.Int
	lastBalanceTime time.Time
	balanceMu       sync.RWMutex
}

// ParallelConfig holds configuration for parallel transactions
type ParallelConfig struct {
	Value                *big.Int
	GasLimit             uint64
	Data                 []byte
	MaxTransactions      int
	MaxConcurrentRequests int    // Maximum concurrent RPC requests
	BalanceCheckInterval int    // Check balance every N transactions
	MaxRetries           int    // Maximum retries for failed transactions
	RetryDelay           time.Duration // Delay between retries
}

// NewParallelSender creates a new parallel transaction sender
func NewParallelSender(client *ethclient.Client, chainID *big.Int, wallets []*ParallelWallet, recipients []common.Address, config *ParallelConfig) *ParallelSender {
	// Set defaults if not provided
	if config.MaxConcurrentRequests == 0 {
		config.MaxConcurrentRequests = 2000
	}
	if config.BalanceCheckInterval == 0 {
		config.BalanceCheckInterval = 100
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 100 * time.Millisecond
	}

	return &ParallelSender{
		client:     client,
		chainID:    chainID,
		wallets:    wallets,
		recipients: recipients,
		config:     config,
		errors:     make([]error, 0),
	}
}

// SendParallelTransactions sends transactions continuously from all wallets until balance runs out
// It respects context cancellation and properly handles errors
func (ps *ParallelSender) SendParallelTransactions(ctx context.Context) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, ps.config.MaxConcurrentRequests)

	// Launch continuous transaction sending from each wallet
	for _, wallet := range ps.wallets {
		wg.Add(1)
		go func(w *ParallelWallet) {
			defer wg.Done()

			rng := rand.New(rand.NewSource(rand.Int63()))
			balanceCheckCounter := 0

			// Continuous loop - send transactions until balance runs out or context is cancelled
			for {
				// Check context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Check balance periodically using cached value when possible
				balanceCheckCounter++
				if balanceCheckCounter%ps.config.BalanceCheckInterval == 0 {
					hasBalance, err := ps.checkWalletBalance(ctx, w)
					if err != nil {
						ps.recordError(fmt.Errorf("wallet %s: balance check failed: %w", w.Address.Hex(), err))
						return
					}
					if !hasBalance {
						return // Wallet out of balance
					}
				}

				// Acquire semaphore (non-blocking)
				select {
				case semaphore <- struct{}{}:
					// Send transaction immediately
					go func() {
						defer func() { <-semaphore }()
						ps.sendTransactionWithRetry(ctx, w, rng)
					}()
				case <-ctx.Done():
					return
				default:
					// Semaphore full, wait a bit before retrying
					select {
					case <-ctx.Done():
						return
					case <-time.After(10 * time.Millisecond):
					}
				}
			}
		}(wallet)
	}

	wg.Wait()

	// Print summary
	ps.printSummary()
	return nil
}

// checkWalletBalance checks if wallet has sufficient balance, using cache when possible
func (ps *ParallelSender) checkWalletBalance(ctx context.Context, w *ParallelWallet) (bool, error) {
	// Check cache first (balance is valid for 1 second)
	w.balanceMu.RLock()
	if w.lastBalance != nil && time.Since(w.lastBalanceTime) < time.Second {
		balance := w.lastBalance
		w.balanceMu.RUnlock()

		gasPrice, err := ps.client.SuggestGasPrice(ctx)
		if err != nil {
			return false, err
		}

		minRequired := new(big.Int).Mul(gasPrice, big.NewInt(int64(ps.config.GasLimit)))
		minRequired.Add(minRequired, ps.config.Value)

		return balance.Cmp(minRequired) >= 0, nil
	}
	w.balanceMu.RUnlock()

	// Cache miss or expired - fetch from network
	balance, err := ps.client.BalanceAt(ctx, w.Address, nil)
	if err != nil {
		return false, err
	}

	gasPrice, err := ps.client.SuggestGasPrice(ctx)
	if err != nil {
		return false, err
	}

	minRequired := new(big.Int).Mul(gasPrice, big.NewInt(int64(ps.config.GasLimit)))
	minRequired.Add(minRequired, ps.config.Value)

	// Update cache
	w.balanceMu.Lock()
	w.lastBalance = balance
	w.lastBalanceTime = time.Now()
	w.balanceMu.Unlock()

	return balance.Cmp(minRequired) >= 0, nil
}

// sendTransactionWithRetry sends a transaction with retry logic
func (ps *ParallelSender) sendTransactionWithRetry(ctx context.Context, w *ParallelWallet, rng *rand.Rand) {
	recipient := ps.recipients[rng.Intn(len(ps.recipients))]

	var lastErr error
	for attempt := 0; attempt <= ps.config.MaxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Get nonce
		nonce, err := w.NonceManager.GetNextNonce(ctx)
		if err != nil {
			lastErr = fmt.Errorf("failed to get nonce: %w", err)
			ps.recordError(fmt.Errorf("wallet %s: %w", w.Address.Hex(), lastErr))
			atomic.AddInt64(&ps.totalFailed, 1)
			return
		}

		// Get gas price
		gasPrice, err := ps.client.SuggestGasPrice(ctx)
		if err != nil {
			lastErr = fmt.Errorf("failed to get gas price: %w", err)
			if attempt < ps.config.MaxRetries {
				time.Sleep(ps.config.RetryDelay * time.Duration(attempt+1))
				continue
			}
			ps.recordError(fmt.Errorf("wallet %s: %w", w.Address.Hex(), lastErr))
			atomic.AddInt64(&ps.totalFailed, 1)
			return
		}

		// Create transaction
		tx := types.NewTransaction(
			nonce,
			recipient,
			ps.config.Value,
			ps.config.GasLimit,
			gasPrice,
			ps.config.Data,
		)

		// Sign transaction
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ps.chainID), w.PrivateKey)
		if err != nil {
			lastErr = fmt.Errorf("failed to sign transaction: %w", err)
			ps.recordError(fmt.Errorf("wallet %s: %w", w.Address.Hex(), lastErr))
			atomic.AddInt64(&ps.totalFailed, 1)
			return
		}

		// Send transaction
		err = ps.client.SendTransaction(ctx, signedTx)
		if err != nil {
			lastErr = fmt.Errorf("failed to send transaction: %w", err)
			if attempt < ps.config.MaxRetries {
				// Retry with exponential backoff
				time.Sleep(ps.config.RetryDelay * time.Duration(attempt+1))
				continue
			}
			ps.recordError(fmt.Errorf("wallet %s: %w", w.Address.Hex(), lastErr))
			atomic.AddInt64(&ps.totalFailed, 1)
			return
		}

		// Success - verify transaction was accepted (optional, non-blocking)
		atomic.AddInt64(&ps.totalSent, 1)
		go ps.verifyTransaction(ctx, signedTx.Hash(), w.Address)
		return
	}

	// All retries failed
	ps.recordError(fmt.Errorf("wallet %s: transaction failed after %d retries: %w", w.Address.Hex(), ps.config.MaxRetries, lastErr))
	atomic.AddInt64(&ps.totalFailed, 1)
}

// verifyTransaction verifies that a transaction was accepted into the mempool
func (ps *ParallelSender) verifyTransaction(ctx context.Context, txHash common.Hash, walletAddr common.Address) {
	// Wait a bit for transaction to be accepted
	time.Sleep(500 * time.Millisecond)

	// Check if transaction is pending
	_, isPending, err := ps.client.TransactionByHash(ctx, txHash)
	if err == nil && !isPending {
		// Transaction was mined
		atomic.AddInt64(&ps.totalSucceeded, 1)
	} else if err == nil && isPending {
		// Transaction is pending - consider it successful
		atomic.AddInt64(&ps.totalSucceeded, 1)
	}
	// If error, we don't increment succeeded but also don't fail - transaction might still be processing
}

// recordError records an error (thread-safe)
func (ps *ParallelSender) recordError(err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// Limit error storage to prevent memory issues
	if len(ps.errors) < 1000 {
		ps.errors = append(ps.errors, err)
	}
}

// GetMetrics returns transaction metrics
func (ps *ParallelSender) GetMetrics() (sent, succeeded, failed int64, errors []error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	errorCopy := make([]error, len(ps.errors))
	copy(errorCopy, ps.errors)
	return atomic.LoadInt64(&ps.totalSent), atomic.LoadInt64(&ps.totalSucceeded), atomic.LoadInt64(&ps.totalFailed), errorCopy
}

// printSummary prints a summary of transactions sent
func (ps *ParallelSender) printSummary() {
	sent, succeeded, failed, errors := ps.GetMetrics()
	fmt.Printf("\n=== Transaction Summary ===\n")
	fmt.Printf("Total sent: %d\n", sent)
	fmt.Printf("Succeeded: %d\n", succeeded)
	fmt.Printf("Failed: %d\n", failed)
	if len(errors) > 0 && len(errors) <= 10 {
		fmt.Printf("\nRecent errors:\n")
		for _, err := range errors[len(errors)-10:] {
			fmt.Printf("  - %s\n", err.Error())
		}
	} else if len(errors) > 10 {
		fmt.Printf("\nShowing last 10 of %d errors:\n", len(errors))
		for _, err := range errors[len(errors)-10:] {
			fmt.Printf("  - %s\n", err.Error())
		}
	}
	fmt.Printf("==========================\n")
}
