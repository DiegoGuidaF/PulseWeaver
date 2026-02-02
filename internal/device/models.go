package device

import (
	"fmt"
	"strings"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
)

type Device struct {
	ID        string        `db:"id" json:"id"`
	Name      string        `db:"name" json:"name"`
	CreatedAt database.Time `db:"created_at" json:"created_at"`
}

// CreateDeviceRequest represents the JSON payload for creating a device
type CreateDeviceRequest struct {
	Name string `json:"name"`
}

// Validate checks if the request is valid
func (r *CreateDeviceRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) < 3 {
		return fmt.Errorf("name must be at least 3 characters")
	}
	if len(r.Name) > 255 {
		return fmt.Errorf("name must be at most 255 characters")
	}
	return nil
}
