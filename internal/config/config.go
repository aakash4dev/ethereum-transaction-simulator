package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	RPCURL          string
	PrivateKey      string
	Value           string
	GasLimit        uint64
	TransactionData string
	MaxTransactions int
	DelaySeconds    int
	RetryDelay      int
	Mode            string // "transfer", "deploy", "interact", "all", "parallel"
	MinBalance      string // Minimum balance to create wallets (default: 100000)
	WalletCount     int    // Number of wallets to create (default: 1000)
	FundingAmount   string // Amount to fund each wallet (default: 100)
}

// Load loads configuration from .env file and environment variables with defaults
func Load() *Config {
	// Try to load .env file, ignore error if it doesn't exist
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables and defaults")
	}

	return &Config{
		RPCURL:          getEnv("RPC_URL", "http://127.0.0.1:8545"),
		PrivateKey:      getEnv("PRIVATE_KEY", ""),
		Value:            getEnv("VALUE", "1"),
		GasLimit:         getEnvUint64("GAS_LIMIT", 210000),
		TransactionData: getEnv("TX_DATA", "lets bomb the network with transactions! AMF to the moon : ) 🚀"),
		MaxTransactions:  getEnvInt("MAX_TRANSACTIONS", 10000),
		DelaySeconds:     getEnvInt("DELAY_SECONDS", 1),
		RetryDelay:       getEnvInt("RETRY_DELAY", 10),
		Mode:             getEnv("MODE", "all"),
		MinBalance:        getEnv("MIN_BALANCE", "100000"),
		WalletCount:       getEnvInt("WALLET_COUNT", 1000),
		FundingAmount:     getEnv("FUNDING_AMOUNT", "100"),
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

