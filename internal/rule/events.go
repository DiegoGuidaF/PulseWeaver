package rule

import (
	"context"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
)

type RuleEventType string

const (
	RuleEventTypeEnabled  RuleEventType = "rule_enabled"
	RuleEventTypeDisabled RuleEventType = "rule_disabled"
)

type RuleEvent struct {
	Type       RuleEventType
	DeviceID   device.DeviceID
	RuleType   RuleType
	TTLSeconds *int
	OccurredAt time.Time
}

type RuleObserver interface {
	OnRuleEvent(ctx context.Context, event RuleEvent)
}
