package transaction

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Sender handles Ethereum transaction operations
type Sender struct {
	client      *ethclient.Client
	privateKey  *ecdsa.PrivateKey
	chainID     *big.Int
	config      *SenderConfig
	nonceManager *NonceManager
}

// SenderConfig holds configuration for transaction sending
type SenderConfig struct {
	RandomAddresses  []common.Address
	Value            *big.Int
	GasLimit         uint64
	Data             []byte
	MaxTransactions  int
	DelaySeconds     int
}

// NewSender creates a new transaction sender
func NewSender(rpcURL, privateKeyHex string, config *SenderConfig) (*Sender, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonceManager := NewNonceManager(client, fromAddress)

	return &Sender{
		client:       client,
		privateKey:   privateKey,
		chainID:      chainID,
		config:       config,
		nonceManager: nonceManager,
	}, nil
}

// NewSenderWithNonceManager creates a new transaction sender with a shared nonce manager
func NewSenderWithNonceManager(rpcURL, privateKeyHex string, config *SenderConfig, nonceManager *NonceManager) (*Sender, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	return &Sender{
		client:       client,
		privateKey:   privateKey,
		chainID:      chainID,
		config:       config,
		nonceManager: nonceManager,
	}, nil
}

// SendTransactions sends multiple transactions to random addresses
func (s *Sender) SendTransactions() error {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	ctx := context.Background()

	for i := 0; i < s.config.MaxTransactions; i++ {
		// Select random address from the array
		randomIndex := rng.Intn(len(s.config.RandomAddresses))
		recipient := s.config.RandomAddresses[randomIndex]

		fmt.Printf("Sending transaction %d/%d to %s\n", i+1, s.config.MaxTransactions, recipient.Hex())

		nonce, err := s.nonceManager.GetNextNonce(ctx)
		if err != nil {
			return fmt.Errorf("failed to get nonce: %w", err)
		}

		gasPrice, err := s.client.SuggestGasPrice(context.Background())
		if err != nil {
			return fmt.Errorf("failed to get gas price: %w", err)
		}

		tx := types.NewTransaction(
			nonce,
			recipient,
			s.config.Value,
			s.config.GasLimit,
			gasPrice,
			s.config.Data,
		)

		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(s.chainID), s.privateKey)
		if err != nil {
			return fmt.Errorf("failed to sign transaction: %w", err)
		}

		if err := s.client.SendTransaction(context.Background(), signedTx); err != nil {
			return fmt.Errorf("failed to send transaction: %w", err)
		}

		fmt.Printf("Transaction hash: %s\n", signedTx.Hash().Hex())

		// Wait for transaction to be mined/confirmed before sending next
		if i < s.config.MaxTransactions-1 {
			if s.config.DelaySeconds > 0 {
				// Wait for transaction receipt or use delay as fallback
				receipt, err := s.waitForTransaction(ctx, signedTx.Hash())
				if err != nil {
					// If receipt wait fails, use delay as fallback
					time.Sleep(time.Duration(s.config.DelaySeconds) * time.Second)
				} else if receipt != nil {
					fmt.Printf("Transaction confirmed in block %d\n", receipt.BlockNumber.Uint64())
				}
			} else {
				// No delay configured, still wait for receipt to avoid nonce errors
				s.waitForTransaction(ctx, signedTx.Hash())
			}
		}
	}

	return nil
}

// waitForTransaction waits for a transaction to be mined and returns the receipt
func (s *Sender) waitForTransaction(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for transaction %s", txHash.Hex())
		case <-ticker.C:
			receipt, err := s.client.TransactionReceipt(ctx, txHash)
			if err == nil && receipt != nil {
				return receipt, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Close closes the Ethereum client connection
func (s *Sender) Close() {
	if s.client != nil {
		s.client.Close()
	}
}

