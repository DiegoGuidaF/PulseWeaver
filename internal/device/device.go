package device

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

const APIKeyPrefix = "wdk_"

// DeviceType describes the network behaviour of a device. Keeping it really simple, in the future rules could be created
// on top of it. Main usage now is for filtering.
// For example static devices do not need address lease but mobile do, static ones can have different geoip rules...
type DeviceType string

const (
	DeviceTypeStatic DeviceType = "static"
	DeviceTypeMobile DeviceType = "mobile"
)

// AllowedDeviceTypes is the canonical ordered list of valid device types.
var AllowedDeviceTypes = []DeviceType{
	DeviceTypeStatic,
	DeviceTypeMobile,
}

// DeviceTypeLabels maps each device type to its display label.
var DeviceTypeLabels = map[DeviceType]string{
	DeviceTypeStatic: "Static",
	DeviceTypeMobile: "Mobile",
}

type Device struct {
	ID          DeviceID         `db:"id"`
	Name        string           `db:"name"`
	DeviceType  DeviceType       `db:"device_type"`
	Description *string          `db:"description"`
	Icon        *string          `db:"icon"`
	CreatedAt   time.Time        `db:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at"`
	DeletedAt   *time.Time       `db:"deleted_at"`
	KeyPrefix   *string          `db:"key_prefix"`
	LastSeenAt  *database.DBTime `db:"last_seen_at"`
	OwnerID     auth.UserID      `db:"owner_id"`
}

// Update applies patch inputs to the device in-place, validating each field.
// Fields with a nil pointer are left unchanged. The device is not mutated
// unless all validations pass.
//
//   - name:        nil = keep current; non-nil = rename (validated)
//   - deviceType:  nil = keep current; non-nil = set type (validated)
//   - description: nil = keep current; non-nil ptr to nil = clear; non-nil ptr to value = set
//   - icon:        same semantics as description
func (d *Device) Update(name *string, deviceType *string, description **string, icon **string, ownerID *auth.UserID) error {
	// Validate all fields before mutating any of them.
	var parsedType DeviceType
	if name != nil {
		if len(*name) < 1 || len(*name) > 50 {
			return ErrInvalidDeviceName
		}
	}
	if deviceType != nil {
		dt, err := parseDeviceType(*deviceType)
		if err != nil {
			return err
		}
		parsedType = dt
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
	if deviceType != nil {
		d.DeviceType = parsedType
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

func parseDeviceType(t string) (DeviceType, error) {
	for _, allowed := range AllowedDeviceTypes {
		if DeviceType(t) == allowed {
			return DeviceType(t), nil
		}
	}
	return "", ErrInvalidDeviceType
}

type CreateDeviceParams struct {
	Name    string
	OwnerID auth.UserID
}

type DeviceID int64

func (id DeviceID) Int64() int64 {
	return int64(id)
}

func (id DeviceID) String() string {
	return strconv.FormatInt(int64(id), 10)
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
