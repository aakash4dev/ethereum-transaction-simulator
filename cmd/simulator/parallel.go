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

	// Check balance
	walletManager := wallet.NewManager(client, chainID, fundingAmount)
	hasBalance, balance, err := walletManager.CheckBalance(ctx, fromAddress, minBalance)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to check balance: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("Account balance: %s wei\n", balance.String())
	fmt.Printf("Minimum balance required: %s wei\n", minBalance.String())

	// Prepare wallets list
	allWallets := make([]*wallet.Wallet, 0)

	// Add original wallet
	originalNonceManager := transaction.NewNonceManager(client, fromAddress)
	originalWallet := &wallet.Wallet{
		PrivateKey:   privateKey,
		Address:      fromAddress,
		NonceManager: originalNonceManager,
		Client:       client,
	}
	allWallets = append(allWallets, originalWallet)
	fmt.Printf("Added original wallet: %s\n", fromAddress.Hex())

	// Create additional wallets if balance is sufficient
	if hasBalance {
		fmt.Printf("\nBalance sufficient! Creating %d additional wallets...\n", cfg.WalletCount)
		
		// Generate new wallets
		newWallets := walletManager.GenerateWallets(cfg.WalletCount)
		fmt.Printf("Generated %d new wallets\n", len(newWallets))

		// Fund wallets in parallel
		fmt.Printf("Funding wallets with %s wei each...\n", fundingAmount.String())
		if err := walletManager.FundWallets(ctx, originalWallet, newWallets); err != nil {
			fmt.Printf("Warning: Some wallets failed to fund: %v\n", err)
		} else {
			fmt.Printf("Successfully funded %d wallets\n", len(newWallets))
		}

		// Add new wallets to the list
		allWallets = append(allWallets, newWallets...)
	} else {
		fmt.Printf("Balance insufficient. Using only original wallet.\n")
	}

	fmt.Printf("\nTotal wallets ready: %d\n", len(allWallets))

	// Generate random recipient addresses
	randomAddresses := contract.GenerateRandomAddresses(25)
	fmt.Printf("Generated 25 random recipient addresses\n")

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

	// Send transactions in parallel from all wallets
	fmt.Printf("\nStarting parallel transactions from %d wallets...\n", len(parallelWallets))
	fmt.Printf("Each wallet will send %d transactions\n", cfg.MaxTransactions)
	fmt.Printf("Total transactions: %d\n", len(parallelWallets)*cfg.MaxTransactions)
	fmt.Printf("No delays - maximum throughput mode!\n\n")

	if err := parallelSender.SendParallelTransactions(ctx); err != nil {
		fmt.Printf("Error during parallel transactions: %v\n", err)
	} else {
		fmt.Printf("\nAll parallel transactions completed!\n")
	}
}

