package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"

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

func runTransfer(cfg *config.Config) {
	value, ok := new(big.Int).SetString(cfg.Value, 10)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: invalid value format: %s\n", cfg.Value)
		os.Exit(1)
	}

	randomAddresses := contract.GenerateRandomAddresses(25)
	senderConfig := &transaction.SenderConfig{
		RandomAddresses: randomAddresses,
		Value:            value,
		GasLimit:         cfg.GasLimit,
		Data:             []byte(cfg.TransactionData),
		MaxTransactions:  cfg.MaxTransactions,
		DelaySeconds:     cfg.DelaySeconds,
	}

	sender, err := transaction.NewSender(cfg.RPCURL, cfg.PrivateKey, senderConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create sender: %s\n", err.Error())
		os.Exit(1)
	}
	defer sender.Close()

	if err := sender.SendTransactions(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to send transactions: %s\n", err.Error())
		os.Exit(1)
	}
}

func runDeploy(cfg *config.Config) {
	value, ok := new(big.Int).SetString(cfg.Value, 10)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: invalid value format: %s\n", cfg.Value)
		os.Exit(1)
	}

	// Create shared nonce manager for both deployments and transfers
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to RPC: %s\n", err.Error())
		os.Exit(1)
	}
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse private key: %s\n", err.Error())
		os.Exit(1)
	}
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonceManager := transaction.NewNonceManager(client, fromAddress)

	// Initialize nonce manager
	ctx := context.Background()
	if err := nonceManager.Reset(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize nonce manager: %s\n", err.Error())
		os.Exit(1)
	}

	// Run deployments and transfers in parallel
	var wg sync.WaitGroup
	wg.Add(2)

	// Deploy contracts goroutine (30% of transactions)
	go func() {
		defer wg.Done()
		deployerConfig := &contract.DeployerConfig{
			Value:           value,
			GasLimit:        cfg.GasLimit,
			MaxTransactions: cfg.MaxTransactions * 3 / 10, // 30% for deployments
			DelaySeconds:    cfg.DelaySeconds,
		}

		deployer, err := contract.NewDeployerWithNonceManager(cfg.RPCURL, cfg.PrivateKey, deployerConfig, nonceManager)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create deployer: %s\n", err.Error())
			return
		}
		defer deployer.Close()

		_, err = deployer.DeployContract()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to deploy contracts: %s\n", err.Error())
		}
	}()

	// Send regular transactions goroutine (70% of transactions)
	go func() {
		defer wg.Done()
		randomAddresses := contract.GenerateRandomAddresses(25)
		senderConfig := &transaction.SenderConfig{
			RandomAddresses: randomAddresses,
			Value:            value,
			GasLimit:         cfg.GasLimit,
			Data:             []byte(cfg.TransactionData),
			MaxTransactions:  cfg.MaxTransactions * 7 / 10, // 70% for transfers
			DelaySeconds:     cfg.DelaySeconds,
		}

		sender, err := transaction.NewSenderWithNonceManager(cfg.RPCURL, cfg.PrivateKey, senderConfig, nonceManager)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create sender: %s\n", err.Error())
			return
		}
		defer sender.Close()

		if err := sender.SendTransactions(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to send transactions: %s\n", err.Error())
		}
	}()

	wg.Wait()
}

func runInteract(cfg *config.Config) {
	value, ok := new(big.Int).SetString(cfg.Value, 10)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: invalid value format: %s\n", cfg.Value)
		os.Exit(1)
	}

	// Create shared nonce manager (client will be kept open for nonce manager)
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to RPC: %s\n", err.Error())
		os.Exit(1)
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		client.Close()
		fmt.Fprintf(os.Stderr, "Error: failed to parse private key: %s\n", err.Error())
		os.Exit(1)
	}
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonceManager := transaction.NewNonceManager(client, fromAddress)

	// First deploy contracts
	deployerConfig := &contract.DeployerConfig{
		Value:           value,
		GasLimit:        cfg.GasLimit,
		MaxTransactions: 5, // Deploy a few contracts first
		DelaySeconds:    cfg.DelaySeconds,
	}

	deployer, err := contract.NewDeployerWithNonceManager(cfg.RPCURL, cfg.PrivateKey, deployerConfig, nonceManager)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create deployer: %s\n", err.Error())
		os.Exit(1)
	}
	defer deployer.Close()

	contractAddresses, err := deployer.DeployContract()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to deploy contracts: %s\n", err.Error())
		os.Exit(1)
	}

	// Then interact with them
	interactConfig := &contract.DeployerConfig{
		Value:           value,
		GasLimit:        cfg.GasLimit,
		MaxTransactions: cfg.MaxTransactions,
		DelaySeconds:    cfg.DelaySeconds,
	}

	interactDeployer, err := contract.NewDeployerWithNonceManager(cfg.RPCURL, cfg.PrivateKey, interactConfig, nonceManager)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create interact deployer: %s\n", err.Error())
		os.Exit(1)
	}
	defer interactDeployer.Close()

	if err := interactDeployer.InteractWithContract(contractAddresses); err != nil {
		client.Close()
		fmt.Fprintf(os.Stderr, "Error: failed to interact with contracts: %s\n", err.Error())
		os.Exit(1)
	}

	client.Close()
}

func runAll(cfg *config.Config) {
	value, ok := new(big.Int).SetString(cfg.Value, 10)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: invalid value format: %s\n", cfg.Value)
		os.Exit(1)
	}

	// Create shared nonce manager
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to RPC: %s\n", err.Error())
		os.Exit(1)
	}
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse private key: %s\n", err.Error())
		os.Exit(1)
	}
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonceManager := transaction.NewNonceManager(client, fromAddress)

	// Initialize nonce manager before starting goroutines to avoid race condition
	ctx := context.Background()
	if err := nonceManager.Reset(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize nonce manager: %s\n", err.Error())
		os.Exit(1)
	}

	// Run transfer and deploy in parallel
	var wg sync.WaitGroup
	wg.Add(2)

	// Transfer goroutine
	go func() {
		defer wg.Done()
		randomAddresses := contract.GenerateRandomAddresses(25)
		senderConfig := &transaction.SenderConfig{
			RandomAddresses: randomAddresses,
			Value:            value,
			GasLimit:         cfg.GasLimit,
			Data:             []byte(cfg.TransactionData),
			MaxTransactions:  cfg.MaxTransactions,
			DelaySeconds:     cfg.DelaySeconds,
		}

		sender, err := transaction.NewSenderWithNonceManager(cfg.RPCURL, cfg.PrivateKey, senderConfig, nonceManager)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create sender: %s\n", err.Error())
			return
		}
		defer sender.Close()

		if err := sender.SendTransactions(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to send transactions: %s\n", err.Error())
		}
	}()

	// Deploy goroutine
	go func() {
		defer wg.Done()
		deployerConfig := &contract.DeployerConfig{
			Value:           value,
			GasLimit:        cfg.GasLimit,
			MaxTransactions: cfg.MaxTransactions,
			DelaySeconds:    cfg.DelaySeconds,
		}

		deployer, err := contract.NewDeployerWithNonceManager(cfg.RPCURL, cfg.PrivateKey, deployerConfig, nonceManager)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create deployer: %s\n", err.Error())
			return
		}
		defer deployer.Close()

		_, err = deployer.DeployContract()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to deploy contracts: %s\n", err.Error())
		}
	}()

	wg.Wait()
}

func main() {
	cfg := config.Load()

	if cfg.PrivateKey == "" {
		fmt.Fprintf(os.Stderr, "Error: PRIVATE_KEY is required. Set it in .env file or environment variable.\n")
		os.Exit(1)
	}

	switch strings.ToLower(cfg.Mode) {
	case "parallel":
		runParallel(cfg)
	case "transfer":
		runTransfer(cfg)
	case "deploy":
		runDeploy(cfg)
	case "interact":
		runInteract(cfg)
	case "all":
		runAll(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown mode '%s'. Valid modes: parallel, transfer, deploy, interact, all\n", cfg.Mode)
		os.Exit(1)
	}
}

