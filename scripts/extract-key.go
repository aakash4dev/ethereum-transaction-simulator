package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/extract-key.go <keystore-file>")
		fmt.Println("Example: go run scripts/extract-key.go ./local-node-data/keystore/UTC--*")
		os.Exit(1)
	}

	keystoreFile := os.Args[1]
	jsonBytes, err := os.ReadFile(keystoreFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// For dev mode, password is empty
	key, err := keystore.DecryptKey(jsonBytes, "")
	if err != nil {
		fmt.Printf("Error decrypting key: %v\n", err)
		fmt.Println("Note: Make sure the password is empty for dev mode accounts")
		os.Exit(1)
	}

	privateKey := key.PrivateKey
	privateKeyBytes := privateKey.D.Bytes()
	
	// Pad to 32 bytes if needed
	privateKeyHex := hex.EncodeToString(privateKeyBytes)
	if len(privateKeyHex) < 64 {
		privateKeyHex = fmt.Sprintf("%064s", privateKeyHex)
	}
	
	fmt.Printf("\nAccount Address: %s\n", key.Address.Hex())
	fmt.Printf("Private Key: %s\n\n", privateKeyHex)
	fmt.Println("Add this to your .env file:")
	fmt.Printf("PRIVATE_KEY=%s\n", privateKeyHex)
}

