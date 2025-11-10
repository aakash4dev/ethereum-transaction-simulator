package contract

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
	"github.com/airchains-studio/mvp-bomber/internal/transaction"
)

// Deployer handles smart contract deployment and interaction
type Deployer struct {
	client       *ethclient.Client
	privateKey  *ecdsa.PrivateKey
	chainID     *big.Int
	config      *DeployerConfig
	nonceManager *transaction.NonceManager
}

// DeployerConfig holds configuration for contract operations
type DeployerConfig struct {
	Value            *big.Int
	GasLimit         uint64
	MaxTransactions  int
	DelaySeconds     int
}

// NewDeployer creates a new contract deployer
func NewDeployer(rpcURL, privateKeyHex string, config *DeployerConfig) (*Deployer, error) {
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
	nonceManager := transaction.NewNonceManager(client, fromAddress)

	return &Deployer{
		client:       client,
		privateKey:  privateKey,
		chainID:     chainID,
		config:      config,
		nonceManager: nonceManager,
	}, nil
}

// NewDeployerWithNonceManager creates a new contract deployer with a shared nonce manager
func NewDeployerWithNonceManager(rpcURL, privateKeyHex string, config *DeployerConfig, nonceManager *transaction.NonceManager) (*Deployer, error) {
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

	return &Deployer{
		client:       client,
		privateKey:  privateKey,
		chainID:     chainID,
		config:      config,
		nonceManager: nonceManager,
	}, nil
}

// DeployContract deploys a smart contract multiple times and returns deployed addresses
func (d *Deployer) DeployContract() ([]common.Address, error) {
	fromAddress := crypto.PubkeyToAddress(d.privateKey.PublicKey)
	deployedAddresses := make([]common.Address, 0, d.config.MaxTransactions)
	ctx := context.Background()

	bytecode, err := GetContractBytecode()
	if err != nil {
		return nil, fmt.Errorf("failed to get contract bytecode: %w", err)
	}

	for i := 0; i < d.config.MaxTransactions; i++ {
		fmt.Printf("Deploying contract %d/%d\n", i+1, d.config.MaxTransactions)

		nonce, err := d.nonceManager.GetNextNonce(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce: %w", err)
		}

		// Retry getting gas price in case of transient node errors
		var gasPrice *big.Int
		maxRetries := 3
		for retry := 0; retry < maxRetries; retry++ {
			gasPrice, err = d.client.SuggestGasPrice(context.Background())
			if err == nil {
				break
			}
			if retry < maxRetries-1 {
				// Wait a bit before retrying (exponential backoff)
				time.Sleep(time.Duration(retry+1) * 200 * time.Millisecond)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get gas price after %d retries: %w", maxRetries, err)
		}

		tx := types.NewContractCreation(nonce, d.config.Value, d.config.GasLimit, gasPrice, bytecode)

		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(d.chainID), d.privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign transaction: %w", err)
		}

		if err := d.client.SendTransaction(context.Background(), signedTx); err != nil {
			return nil, fmt.Errorf("failed to send transaction: %w", err)
		}

		// Calculate contract address
		contractAddress := crypto.CreateAddress(fromAddress, nonce)
		deployedAddresses = append(deployedAddresses, contractAddress)

		fmt.Printf("Deployment transaction hash: %s, contract address: %s\n", 
			signedTx.Hash().Hex(), contractAddress.Hex())

		// Wait for the node to accept the transaction into mempool before proceeding
		// This prevents nonce conflicts when sending transactions rapidly
		if i < d.config.MaxTransactions-1 {
			if d.config.DelaySeconds > 0 {
				// Wait for transaction receipt or use delay as fallback
				time.Sleep(time.Duration(d.config.DelaySeconds) * time.Second)
			} else {
				// Wait for nonce to update (node has accepted tx into mempool)
				// This ensures PendingNonceAt will reflect our transaction
				d.nonceManager.WaitForNonceUpdate(ctx, nonce, 2*time.Second)
			}
		}
	}

	return deployedAddresses, nil
}

// InteractWithContract calls a contract function multiple times on deployed contracts
func (d *Deployer) InteractWithContract(contractAddresses []common.Address) error {
	if len(contractAddresses) == 0 {
		return fmt.Errorf("at least one contract address is required for interaction")
	}

	// Generate random value for each function call
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	ctx := context.Background()

	for i := 0; i < d.config.MaxTransactions; i++ {
		// Select random contract address
		contractIndex := rng.Intn(len(contractAddresses))
		contractAddress := contractAddresses[contractIndex]
		
		// Generate random value for the set function
		randomValue := big.NewInt(int64(rng.Intn(1000000) + 1))
		functionData, err := GetSetFunctionData(randomValue)
		if err != nil {
			return fmt.Errorf("failed to generate function data: %w", err)
		}

		fmt.Printf("Calling contract function %d/%d on %s with value %s\n", 
			i+1, d.config.MaxTransactions, contractAddress.Hex(), randomValue.String())

		nonce, err := d.nonceManager.GetNextNonce(ctx)
		if err != nil {
			return fmt.Errorf("failed to get nonce: %w", err)
		}

		// Retry getting gas price in case of transient node errors
		var gasPrice *big.Int
		maxRetries := 3
		for retry := 0; retry < maxRetries; retry++ {
			gasPrice, err = d.client.SuggestGasPrice(context.Background())
			if err == nil {
				break
			}
			if retry < maxRetries-1 {
				// Wait a bit before retrying (exponential backoff)
				time.Sleep(time.Duration(retry+1) * 200 * time.Millisecond)
			}
		}
		if err != nil {
			return fmt.Errorf("failed to get gas price after %d retries: %w", maxRetries, err)
		}

		tx := types.NewTransaction(
			nonce,
			contractAddress,
			d.config.Value,
			d.config.GasLimit,
			gasPrice,
			functionData,
		)

		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(d.chainID), d.privateKey)
		if err != nil {
			return fmt.Errorf("failed to sign transaction: %w", err)
		}

		if err := d.client.SendTransaction(context.Background(), signedTx); err != nil {
			return fmt.Errorf("failed to send transaction: %w", err)
		}

		fmt.Printf("Interaction transaction hash: %s\n", signedTx.Hash().Hex())

		if i < d.config.MaxTransactions-1 {
			time.Sleep(time.Duration(d.config.DelaySeconds) * time.Second)
		}
	}

	return nil
}

// Close closes the Ethereum client connection
func (d *Deployer) Close() {
	if d.client != nil {
		d.client.Close()
	}
}

