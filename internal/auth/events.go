package auth

import (
	"context"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type UserEventType string

const (
	EventTypeUserCreated UserEventType = "user_created"
	EventTypeUserDeleted UserEventType = "user_deleted"
)

type UserEvent struct {
	Type   UserEventType
	UserID ids.UserID
}

// UserObserver is implemented by any service that needs to react to user lifecycle events.
// Implementations are called synchronously within the originating transaction context.
type UserObserver interface {
	OnUserEvent(ctx context.Context, event UserEvent)
}
