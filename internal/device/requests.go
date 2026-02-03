package device

import (
	"errors"
	"fmt"
	"strings"
)

// CreateDeviceRequest represents the JSON payload for creating a device
type CreateDeviceRequest struct {
	Name string `json:"name" example:"Personal phone" binding:"required"`
}

// Validate checks if the request is valid
func (r *CreateDeviceRequest) Validate() error {
	name := strings.TrimSpace(r.Name)

	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) < 3 {
		return fmt.Errorf("name must be at least 3 characters")
	}
	if len(name) > 255 {
		return fmt.Errorf("name must be at most 255 characters")
	}
	return nil
}

type AssignDeviceIPRequest struct {
	IPAddress string `json:"ip_address" example:"192.168.1.132" binding:"required"`
}

func (r *AssignDeviceIPRequest) Validate() error {
	if r.IPAddress == "" {
		return errors.New("ip_address is required")
	}
	return nil
}
