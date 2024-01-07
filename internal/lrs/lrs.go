package lrs

import (
	"fmt"
	"github.com/zbohm/lirisi/client"
	"github.com/zbohm/lirisi/ring"
)

// FoldPublicKeys is a wrapper for client.FoldPublicKeys
func FoldPublicKeys(publicKeys []string) (string, error) {
	// Convert public keys to byte arrays.
	var publicKeysContent [][]byte
	for _, key := range publicKeys {
		publicKeysContent = append(publicKeysContent, []byte(key))
	}

	// Fold public keys.
	status, foldedPublicKeys := client.FoldPublicKeys(publicKeysContent, "sha3-256", "PEM", "hashes")
	if status != ring.Success {
		return "", fmt.Errorf("client.FoldPublicKeys() failed: %v", status)
	}

	return string(foldedPublicKeys), nil
}
