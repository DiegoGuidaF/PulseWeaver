package lease

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// AddressDisabler disables a set of addresses from a given event source.
type AddressDisabler interface {
	DisableAddresses(ctx context.Context, addressIDs []ids.AddressID, source device.EventSource) error
}

// ExpiryJob disables addresses whose time-based lease has expired.
type ExpiryJob struct {
	service  *Service
	disabler AddressDisabler
	logger   *slog.Logger
}

// NewExpiryJob returns a scheduled job that expires addresses via this service.
func (s *Service) NewExpiryJob(disabler AddressDisabler) *ExpiryJob {
	return &ExpiryJob{
		service:  s,
		disabler: disabler,
		logger:   s.logger.With(slog.String(logging.AttrKeyComponent, "lease_expiry_job")),
	}
}

func (j *ExpiryJob) Run(ctx context.Context) error {
	j.logger.InfoContext(ctx, "starting auto-expiry task")

	ids, err := j.service.GetExpiredAddressIDs(ctx)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		j.logger.DebugContext(ctx, "no expired addresses detected")
		return nil
	}
	j.logger.InfoContext(ctx, "expired addresses detected", slog.Int(logging.AttrKeyCount, len(ids)))

	return j.disabler.DisableAddresses(ctx, ids, device.EventSourceExpiry)
}
