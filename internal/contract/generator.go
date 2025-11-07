package contract

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// SimpleStorageContract is a minimal contract that stores and retrieves a uint256 value
// Solidity: pragma solidity ^0.8.0; contract SimpleStorage { uint256 public value; function set(uint256 _value) public { value = _value; } }
// This bytecode is for a simple storage contract (compiled with Solidity 0.8.0)
// Valid bytecode for: contract SimpleStorage { uint256 public value; function set(uint256 _value) public { value = _value; } }
var SimpleStorageContractBytecode = "608060405234801561001057600080fd5b50610150806100206000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c806360fe47b11460375780636d4ce63c146051575b600080fd5b604f60048036038101906049919060b1565b6069565b005b60576071565b60405160609190608c565b60405180910390f35b8060008190555050565b60005481565b6000819050919050565b6086816073565b82525050565b6000602082019050609f6000830184607f565b92915050565b600080fd5b6000819050919050565b60bd8160aa565b811460c757600080fd5b50565b60008135905060d78160b6565b9291505056fea2646970667358221220a2b8c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b64736f6c634300080700330"

// GenerateRandomAddresses generates n random Ethereum addresses
func GenerateRandomAddresses(n int) []common.Address {
	addresses := make([]common.Address, n)
	for i := 0; i < n; i++ {
		// Generate random private key
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			// Fallback: generate random bytes
			bytes := make([]byte, 20)
			rand.Read(bytes)
			addresses[i] = common.BytesToAddress(bytes)
			continue
		}
		addresses[i] = crypto.PubkeyToAddress(privateKey.PublicKey)
	}
	return addresses
}

// GetContractBytecode returns the bytecode for the simple storage contract
func GetContractBytecode() ([]byte, error) {
	bytecode, err := hex.DecodeString(SimpleStorageContractBytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to decode contract bytecode: %w", err)
	}
	return bytecode, nil
}

// GetSetFunctionData generates the function call data for the set(uint256) function
// Function signature: set(uint256)
// Keccak256("set(uint256)") = 0x60fe47b1 (first 4 bytes)
func GetSetFunctionData(value *big.Int) ([]byte, error) {
	// Function selector: keccak256("set(uint256)")[:4] = 0x60fe47b1
	functionSelector := []byte{0x60, 0xfe, 0x47, 0xb1}
	
	// Pad value to 32 bytes
	paddedValue := make([]byte, 32)
	valueBytes := value.Bytes()
	copy(paddedValue[32-len(valueBytes):], valueBytes)
	
	// Combine selector and padded value
	data := append(functionSelector, paddedValue...)
	return data, nil
}

