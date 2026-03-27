// Package config - encrypted configuration archive for export/import across machines.
// Bundles profiles, config, and proxy lists into a single encrypted JSON file.
package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Archive is the portable export format containing all user data.
type Archive struct {
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"createdAt"`
	Config    *AppConfig      `json:"config"`
	Profiles  json.RawMessage `json:"profiles,omitempty"`
	Proxies   []string        `json:"proxies,omitempty"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
}

// ExportArchive creates an encrypted archive of the current configuration.
func ExportArchive(cfg *AppConfig, profiles json.RawMessage, proxies []string, passphrase string) ([]byte, error) {
	archive := Archive{
		Version:   1,
		CreatedAt: time.Now(),
		Config:    cfg,
		Profiles:  profiles,
		Proxies:   proxies,
		Metadata: map[string]any{
			"app":     "mongebot",
			"version": "2.0.0",
		},
	}

	plaintext, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling archive: %w", err)
	}

	if passphrase == "" {
		// No encryption — return plaintext
		return plaintext, nil
	}

	return encryptArchive(plaintext, passphrase)
}

// ImportArchive decrypts and parses an archive file.
func ImportArchive(data []byte, passphrase string) (*Archive, error) {
	var plaintext []byte

	// Try parsing as plain JSON first
	var archive Archive
	if err := json.Unmarshal(data, &archive); err == nil && archive.Version > 0 {
		return &archive, nil
	}

	// Try decrypting
	if passphrase == "" {
		return nil, fmt.Errorf("archive appears encrypted but no passphrase provided")
	}

	var err error
	plaintext, err = decryptArchive(data, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong passphrase?): %w", err)
	}

	if err := json.Unmarshal(plaintext, &archive); err != nil {
		return nil, fmt.Errorf("parsing decrypted archive: %w", err)
	}

	return &archive, nil
}

// ExportToFile creates an archive and writes it to a file.
func ExportToFile(path string, cfg *AppConfig, profiles json.RawMessage, proxies []string, passphrase string) error {
	data, err := ExportArchive(cfg, profiles, proxies, passphrase)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// ImportFromFile reads and parses an archive from a file.
func ImportFromFile(path string, passphrase string) (*Archive, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading archive: %w", err)
	}
	return ImportArchive(data, passphrase)
}

// encryptArchive encrypts data with AES-256-GCM using a SHA-256 key derived from passphrase.
func encryptArchive(plaintext []byte, passphrase string) ([]byte, error) {
	key := sha256.Sum256([]byte(passphrase))

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decryptArchive decrypts data with AES-256-GCM.
func decryptArchive(ciphertext []byte, passphrase string) ([]byte, error) {
	key := sha256.Sum256([]byte(passphrase))

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
