package config

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	RPCURL                string
	PrivateKey            string
	Value                 string
	GasLimit              uint64
	TransactionData       string
	MaxTransactions       int
	DelaySeconds          int
	RetryDelay            int
	Mode                  string // "transfer", "deploy", "interact", "all", "parallel"
	MinBalance            string // Minimum balance to create wallets (default: 100000)
	WalletCount           int    // Number of wallets to create (default: 1000)
	FundingAmount         string // Amount to fund each wallet (default: 100)
	MaxConcurrentRequests int    // Maximum concurrent RPC requests (default: 2000)
	BalanceCheckInterval  int    // Check balance every N transactions (default: 100)
	FundingConcurrency    int    // Concurrent funding operations (default: 50)
}

// Load loads configuration from .env file and environment variables with defaults
func Load() *Config {
	// Try to load .env file, ignore error if it doesn't exist
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables and defaults")
	}

	return &Config{
		RPCURL:                getEnv("RPC_URL", "http://127.0.0.1:8545"),
		PrivateKey:            getEnv("PRIVATE_KEY", ""),
		Value:                 getEnv("VALUE", "1"),
		GasLimit:              getEnvUint64("GAS_LIMIT", 210000),
		TransactionData:       getEnv("TX_DATA", "lets bomb the network with transactions! AMF to the moon : ) 🚀"),
		MaxTransactions:       getEnvInt("MAX_TRANSACTIONS", 10000),
		DelaySeconds:          getEnvInt("DELAY_SECONDS", 1),
		RetryDelay:            getEnvInt("RETRY_DELAY", 10),
		Mode:                  getEnv("MODE", "all"),
		MinBalance:            getEnv("MIN_BALANCE", "100000"),
		WalletCount:           getEnvInt("WALLET_COUNT", 1000),
		FundingAmount:         getEnv("FUNDING_AMOUNT", "100"),
		MaxConcurrentRequests: getEnvInt("MAX_CONCURRENT_REQUESTS", 2000),
		BalanceCheckInterval:  getEnvInt("BALANCE_CHECK_INTERVAL", 100),
		FundingConcurrency:    getEnvInt("FUNDING_CONCURRENCY", 50),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvUint64(key string, defaultValue uint64) uint64 {
	if value := os.Getenv(key); value != "" {
		if uintValue, err := strconv.ParseUint(value, 10, 64); err == nil {
			return uintValue
		}
	}
	return defaultValue
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	// Validate private key
	if c.PrivateKey == "" {
		return errors.New("PRIVATE_KEY is required")
	}
	
	// Remove 0x prefix if present
	privateKeyHex := strings.TrimPrefix(c.PrivateKey, "0x")
	
	// Validate private key format (should be 64 hex characters)
	if len(privateKeyHex) != 64 {
		return fmt.Errorf("PRIVATE_KEY must be 64 hex characters (got %d)", len(privateKeyHex))
	}
	
	// Try to parse private key to ensure it's valid
	_, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return fmt.Errorf("PRIVATE_KEY is invalid: %w", err)
	}
	
	// Validate RPC URL
	if c.RPCURL == "" {
		return errors.New("RPC_URL is required")
	}
	if !strings.HasPrefix(c.RPCURL, "http://") && !strings.HasPrefix(c.RPCURL, "https://") && !strings.HasPrefix(c.RPCURL, "ws://") && !strings.HasPrefix(c.RPCURL, "wss://") {
		return fmt.Errorf("RPC_URL must start with http://, https://, ws://, or wss://")
	}
	
	// Validate mode
	validModes := map[string]bool{
		"parallel": true,
		"transfer": true,
		"deploy":   true,
		"interact": true,
		"all":      true,
	}
	if !validModes[strings.ToLower(c.Mode)] {
		return fmt.Errorf("MODE must be one of: parallel, transfer, deploy, interact, all (got: %s)", c.Mode)
	}
	
	// Validate value (must be a valid number)
	value, ok := new(big.Int).SetString(c.Value, 10)
	if !ok {
		return fmt.Errorf("VALUE must be a valid number (got: %s)", c.Value)
	}
	if value.Sign() < 0 {
		return errors.New("VALUE cannot be negative")
	}
	
	// Validate gas limit
	if c.GasLimit == 0 {
		return errors.New("GAS_LIMIT must be greater than 0")
	}
	if c.GasLimit > 30000000 { // Ethereum block gas limit is around 30M
		return fmt.Errorf("GAS_LIMIT is too high (max: 30000000, got: %d)", c.GasLimit)
	}
	
	// Validate max transactions
	if c.MaxTransactions < 0 {
		return errors.New("MAX_TRANSACTIONS cannot be negative")
	}
	
	// Validate delay seconds
	if c.DelaySeconds < 0 {
		return errors.New("DELAY_SECONDS cannot be negative")
	}
	
	// Validate min balance
	minBalance, ok := new(big.Int).SetString(c.MinBalance, 10)
	if !ok {
		return fmt.Errorf("MIN_BALANCE must be a valid number (got: %s)", c.MinBalance)
	}
	if minBalance.Sign() < 0 {
		return errors.New("MIN_BALANCE cannot be negative")
	}
	
	// Validate wallet count
	if c.WalletCount < 0 {
		return errors.New("WALLET_COUNT cannot be negative")
	}
	if c.WalletCount > 10000 {
		return fmt.Errorf("WALLET_COUNT is too high (max: 10000, got: %d)", c.WalletCount)
	}
	
	// Validate funding amount
	fundingAmount, ok := new(big.Int).SetString(c.FundingAmount, 10)
	if !ok {
		return fmt.Errorf("FUNDING_AMOUNT must be a valid number (got: %s)", c.FundingAmount)
	}
	if fundingAmount.Sign() < 0 {
		return errors.New("FUNDING_AMOUNT cannot be negative")
	}
	
	// Validate max concurrent requests
	if c.MaxConcurrentRequests <= 0 {
		return errors.New("MAX_CONCURRENT_REQUESTS must be greater than 0")
	}
	if c.MaxConcurrentRequests > 10000 {
		return fmt.Errorf("MAX_CONCURRENT_REQUESTS is too high (max: 10000, got: %d)", c.MaxConcurrentRequests)
	}
	
	// Validate balance check interval
	if c.BalanceCheckInterval <= 0 {
		return errors.New("BALANCE_CHECK_INTERVAL must be greater than 0")
	}
	
	// Validate funding concurrency
	if c.FundingConcurrency <= 0 {
		return errors.New("FUNDING_CONCURRENCY must be greater than 0")
	}
	if c.FundingConcurrency > 1000 {
		return fmt.Errorf("FUNDING_CONCURRENCY is too high (max: 1000, got: %d)", c.FundingConcurrency)
	}
	
	return nil
}

