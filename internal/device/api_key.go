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

const ApiKeyPrefix = "wdk_"

type ApiKey struct {
	ID        ApiKeyID  `db:"id"`
	DeviceID  DeviceID  `db:"device_id"`
	KeyPrefix string    `db:"key_prefix"`
	KeyHash   string    `db:"key_hash"`
	CreatedAt time.Time `db:"created_at"`
}

type ApiKeyID int64

func (id ApiKeyID) Int64() int64 {
	return int64(id)
}

func (id ApiKeyID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// generateApiKey generates a new API key and returns the raw key (to send to user),
// the key hash (to store in DB), and the key prefix (for display).
func generateApiKey() (rawKey string, keyHash string, keyPrefix string, error error) {
	// Generate 32 random bytes (same as auth tokens)
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", "", "", fmt.Errorf("generate random bytes: %w", err)
	}

	// Encode as URL-safe base64, no padding
	rawKey = base64.RawURLEncoding.EncodeToString(b)

	// Add prefix for easy identification
	prefixedKey := ApiKeyPrefix + rawKey

	// Hash for storage (SHA-256, same pattern as auth tokens)
	keyHash = hashApiKey(prefixedKey)

	// Extract prefix for display (first 8 chars after prefix)
	keyPrefix = prefixedKey[:len(ApiKeyPrefix)+8]

	return prefixedKey, keyHash, keyPrefix, nil
}

// hashApiKey hashes the API key using SHA-256 and returns base64url encoded hash.
// This mirrors the pattern used in auth.hashRawToken.
func hashApiKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// NewApiKey creates a new ApiKey domain entity for a device.
func NewApiKey(deviceID DeviceID) (*ApiKey, string, error) {
	rawKey, keyHash, keyPrefix, err := generateApiKey()
	if err != nil {
		return nil, "", err
	}

	return &ApiKey{
		DeviceID:  deviceID,
		KeyPrefix: keyPrefix,
		KeyHash:   keyHash,
		CreatedAt: time.Now().UTC(),
	}, rawKey, nil
}
