// Package fingerprint generates browser fingerprints and device IDs.
package fingerprint

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateDeviceID creates a random 32-character alphanumeric device ID.
func GenerateDeviceID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateNonce creates a random nonce of the specified length.
func GenerateNonce(length int) string {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
