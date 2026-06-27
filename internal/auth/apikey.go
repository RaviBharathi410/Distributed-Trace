package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func GenerateAPIKey(isProd bool) (string, string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", "", err
	}
	
	keyString := hex.EncodeToString(bytes)
	prefix := "dt_test_"
	if isProd {
		prefix = "dt_live_"
	}
	
	fullKey := prefix + keyString
	hash := HashAPIKey(fullKey)
	return fullKey, hash, nil
}

func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
