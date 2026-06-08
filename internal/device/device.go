package device

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

const APIKeyPrefix = "wdk_"

type Device struct {
	ID          ids.DeviceID `db:"id"`
	Name        string       `db:"name"`
	Description *string      `db:"description"`
	Icon        *string      `db:"icon"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"`
	DeletedAt   *time.Time   `db:"deleted_at"`
	DisabledAt  *time.Time   `db:"disabled_at"`
	KeyPrefix   *string      `db:"key_prefix"`
	OwnerID     ids.UserID   `db:"owner_id"`
}

// Update applies patch inputs to the device in-place, validating each field.
// Fields with a nil pointer are left unchanged. The device is not mutated
// unless all validations pass.
//
//   - name:        nil = keep current; non-nil = rename (validated)
//   - description: nil = keep current; non-nil ptr to nil = clear; non-nil ptr to value = set
//   - icon:        same semantics as description
func (d *Device) Update(name *string, description **string, icon **string, ownerID *ids.UserID) error {
	// Validate all fields before mutating any of them.
	if name != nil {
		if len(*name) < 1 || len(*name) > 50 {
			return ErrInvalidDeviceName
		}
	}
	if description != nil && *description != nil && len(**description) > 200 {
		return ErrDescriptionTooLong
	}
	if icon != nil && *icon != nil && len(**icon) > 80 {
		return ErrIconTooLong
	}

	// All validations passed — apply mutations.
	if name != nil {
		d.Name = *name
	}
	if description != nil {
		d.Description = *description
	}
	if icon != nil {
		d.Icon = *icon
	}
	if ownerID != nil {
		d.OwnerID = *ownerID
	}

	return nil
}

type CreateDeviceParams struct {
	Name        string
	OwnerID     ids.UserID
	Description *string
	Icon        *string
}

// GenerateAPIKey generates a new API key and returns the raw key (to send to user),
// the key hash (to store in DB), and the key prefix (for display).
func GenerateAPIKey() (rawKey string, keyHash string, keyPrefix string, error error) {
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
	keyHash = HashAPIKey(prefixedKey)

	// Extract prefix for display (first 8 chars after prefix)
	keyPrefix = prefixedKey[:len(APIKeyPrefix)+8]

	return prefixedKey, keyHash, keyPrefix, nil
}

// HashAPIKey hashes the API key using SHA-256 and returns base64url encoded hash.
// This mirrors the pattern used in auth.hashRawToken.
func HashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
