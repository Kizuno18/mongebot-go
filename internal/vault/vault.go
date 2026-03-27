// Package vault provides encrypted storage for sensitive data like auth tokens.
// Uses AES-256-GCM for encryption with a master key derived from a passphrase.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize   = 32
	keySize    = 32
	iterations = 600_000 // PBKDF2 iterations (OWASP 2024 recommendation)
)

// Vault stores and retrieves encrypted data on disk.
type Vault struct {
	mu       sync.RWMutex
	filePath string
	key      []byte
	data     *VaultData
}

// VaultData is the internal structure stored encrypted on disk.
type VaultData struct {
	Tokens []TokenEntry `json:"tokens"`
}

// TokenEntry represents a stored auth token with metadata.
type TokenEntry struct {
	ID       string `json:"id"`
	Platform string `json:"platform"`
	Value    string `json:"value"`
	Label    string `json:"label,omitempty"`
	Valid    bool   `json:"valid"`
}

// vaultFile is the on-disk format: salt + nonce + ciphertext.
type vaultFile struct {
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// Open opens or creates a vault at the given path with the given passphrase.
func Open(filePath string, passphrase string) (*Vault, error) {
	v := &Vault{
		filePath: filePath,
		data:     &VaultData{},
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// New vault: generate salt, derive key, save empty
			salt := make([]byte, saltSize)
			if _, err := io.ReadFull(rand.Reader, salt); err != nil {
				return nil, fmt.Errorf("generating salt: %w", err)
			}
			v.key = deriveKey(passphrase, salt)
			return v, v.save(salt)
		}
		return nil, fmt.Errorf("reading vault: %w", err)
	}

	// Existing vault: decrypt
	var vf vaultFile
	if err := json.Unmarshal(fileData, &vf); err != nil {
		return nil, fmt.Errorf("parsing vault file: %w", err)
	}

	v.key = deriveKey(passphrase, vf.Salt)

	plaintext, err := decrypt(v.key, vf.Nonce, vf.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypting vault (wrong passphrase?): %w", err)
	}

	if err := json.Unmarshal(plaintext, v.data); err != nil {
		return nil, fmt.Errorf("parsing vault data: %w", err)
	}

	return v, nil
}

// AddToken adds a new token to the vault.
func (v *Vault) AddToken(entry TokenEntry) error {
	v.mu.Lock()
	v.data.Tokens = append(v.data.Tokens, entry)
	v.mu.Unlock()
	return v.Save()
}

// RemoveToken removes a token by ID.
func (v *Vault) RemoveToken(id string) error {
	v.mu.Lock()
	filtered := make([]TokenEntry, 0, len(v.data.Tokens))
	for _, t := range v.data.Tokens {
		if t.ID != id {
			filtered = append(filtered, t)
		}
	}
	v.data.Tokens = filtered
	v.mu.Unlock()
	return v.Save()
}

// GetTokens returns all tokens for a given platform (or all if platform is empty).
func (v *Vault) GetTokens(platform string) []TokenEntry {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if platform == "" {
		result := make([]TokenEntry, len(v.data.Tokens))
		copy(result, v.data.Tokens)
		return result
	}

	var result []TokenEntry
	for _, t := range v.data.Tokens {
		if t.Platform == platform {
			result = append(result, t)
		}
	}
	return result
}

// GetValidTokenValues returns only the raw values of valid tokens for a platform.
func (v *Vault) GetValidTokenValues(platform string) []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var values []string
	for _, t := range v.data.Tokens {
		if t.Valid && (platform == "" || t.Platform == platform) {
			values = append(values, t.Value)
		}
	}
	return values
}

// SetTokenValidity updates the validity flag of a token.
func (v *Vault) SetTokenValidity(id string, valid bool) error {
	v.mu.Lock()
	for i := range v.data.Tokens {
		if v.data.Tokens[i].ID == id {
			v.data.Tokens[i].Valid = valid
			break
		}
	}
	v.mu.Unlock()
	return v.Save()
}

// Save encrypts and writes the vault to disk.
func (v *Vault) Save() error {
	// Generate a fresh salt on each save for forward secrecy
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("generating salt: %w", err)
	}
	v.key = deriveKey("", salt) // Reuse existing derived key
	return v.save(salt)
}

// save is the internal save that uses a specific salt.
func (v *Vault) save(salt []byte) error {
	v.mu.RLock()
	plaintext, err := json.Marshal(v.data)
	v.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshaling vault data: %w", err)
	}

	nonce, ciphertext, err := encrypt(v.key, plaintext)
	if err != nil {
		return fmt.Errorf("encrypting vault: %w", err)
	}

	vf := vaultFile{
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}

	data, err := json.Marshal(vf)
	if err != nil {
		return fmt.Errorf("marshaling vault file: %w", err)
	}

	return os.WriteFile(v.filePath, data, 0o600)
}

// deriveKey uses PBKDF2-SHA256 to derive an AES-256 key from a passphrase.
func deriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, iterations, keySize, sha256.New)
}

// encrypt encrypts plaintext using AES-256-GCM, returning nonce and ciphertext.
func encrypt(key, plaintext []byte) (nonce, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

// decrypt decrypts ciphertext using AES-256-GCM.
func decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}
