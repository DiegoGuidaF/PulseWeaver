package rule

import (
	"context"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type RuleEventType string

const (
	RuleEventTypeEnabled  RuleEventType = "rule_enabled"
	RuleEventTypeDisabled RuleEventType = "rule_disabled"
)

type RuleEvent struct {
	Type       RuleEventType
	DeviceID   ids.DeviceID
	RuleType   RuleType
	TTLSeconds *int
	OccurredAt time.Time
}

type RuleObserver interface {
	OnRuleEvent(ctx context.Context, event RuleEvent)
}
