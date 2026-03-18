package device

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"time"
)

const APIKeyPrefix = "wdk_"

type Device struct {
	ID        DeviceID   `db:"id" `
	Name      string     `db:"name" `
	CreatedAt time.Time  `db:"created_at" `
	DeletedAt *time.Time `db:"deleted_at" `
	KeyPrefix string     `db:"key_prefix"`
}

type CreateDeviceParams struct {
	Name      string
	KeyPrefix string
	KeyHash   string
}

func NewCreateDeviceParams(name string) (CreateDeviceParams, string, error) {
	rawKey, keyHash, keyPrefix, err := generateAPIKey()
	if err != nil {
		return CreateDeviceParams{}, "", err
	}
	return CreateDeviceParams{
		Name:      name,
		KeyPrefix: keyPrefix,
		KeyHash:   keyHash,
	}, rawKey, nil
}

type DeviceID int64

func (id DeviceID) Int64() int64 {
	return int64(id)
}

func (id DeviceID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// generateAPIKey generates a new API key and returns the raw key (to send to user),
// the key hash (to store in DB), and the key prefix (for display).
func generateAPIKey() (rawKey string, keyHash string, keyPrefix string, error error) {
	// Generate 32 random bytes (same as auth tokens)
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", "", "", fmt.Errorf("generate random bytes: %w", err)
	}

	// Encode as URL-safe base64, no padding
	rawKey = base64.RawURLEncoding.EncodeToString(b)

	// Add prefix for easy identification
	prefixedKey := APIKeyPrefix + rawKey

	// Hash for storage (SHA-256, same pattern as auth tokens)
	keyHash = hashAPIKey(prefixedKey)

	// Extract prefix for display (first 8 chars after prefix)
	keyPrefix = prefixedKey[:len(APIKeyPrefix)+8]

	return prefixedKey, keyHash, keyPrefix, nil
}

// hashAPIKey hashes the API key using SHA-256 and returns base64url encoded hash.
// This mirrors the pattern used in auth.hashRawToken.
func hashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
