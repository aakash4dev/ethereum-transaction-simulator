package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/airchains-studio/mvp-bomber/internal/config"
	"github.com/airchains-studio/mvp-bomber/internal/contract"
	"github.com/airchains-studio/mvp-bomber/internal/transaction"
	"github.com/airchains-studio/mvp-bomber/internal/wallet"
)

func runParallel(cfg *config.Config) {
	ctx := context.Background()

	// Parse values
	value, ok := new(big.Int).SetString(cfg.Value, 10)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: invalid value format: %s\n", cfg.Value)
		os.Exit(1)
	}

	minBalance, ok := new(big.Int).SetString(cfg.MinBalance, 10)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: invalid min balance format: %s\n", cfg.MinBalance)
		os.Exit(1)
	}

	fundingAmount, ok := new(big.Int).SetString(cfg.FundingAmount, 10)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: invalid funding amount format: %s\n", cfg.FundingAmount)
		os.Exit(1)
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse private key: %s\n", err.Error())
		os.Exit(1)
	}
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create client
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to RPC: %s\n", err.Error())
		os.Exit(1)
	}
	defer client.Close()

	// Get chain ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get chain ID: %s\n", err.Error())
		os.Exit(1)
	}

	walletManager := wallet.NewManager(client, chainID, fundingAmount)
	hasBalance, _, err := walletManager.CheckBalance(ctx, fromAddress, minBalance)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to check balance: %s\n", err.Error())
		os.Exit(1)
	}

	allWallets := make([]*wallet.Wallet, 0)
	originalNonceManager := transaction.NewNonceManager(client, fromAddress)
	originalWallet := &wallet.Wallet{
		PrivateKey:   privateKey,
		Address:      fromAddress,
		NonceManager: originalNonceManager,
		Client:       client,
	}
	allWallets = append(allWallets, originalWallet)

	if hasBalance {
		newWallets := walletManager.GenerateWallets(cfg.WalletCount)
		walletManager.FundWallets(ctx, originalWallet, newWallets)
		allWallets = append(allWallets, newWallets...)
	}

	randomAddresses := contract.GenerateRandomAddresses(25)

	// Convert wallets to parallel wallet format
	parallelWallets := make([]*transaction.ParallelWallet, len(allWallets))
	for i, w := range allWallets {
		parallelWallets[i] = &transaction.ParallelWallet{
			PrivateKey:   w.PrivateKey,
			Address:      w.Address,
			NonceManager: w.NonceManager,
		}
	}

	// Create parallel sender
	parallelConfig := &transaction.ParallelConfig{
		Value:           value,
		GasLimit:        cfg.GasLimit,
		Data:            []byte(cfg.TransactionData),
		MaxTransactions: cfg.MaxTransactions,
	}

	parallelSender := transaction.NewParallelSender(client, chainID, parallelWallets, randomAddresses, parallelConfig)
	parallelSender.SendParallelTransactions(ctx)
}

