package scheduler

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
)

func (s *Service) executeAutoExpiry(ctx context.Context) error {
	s.logger.InfoContext(ctx, "starting auto-expiry task")

	ids, err := s.expiredAddressFinder.GetExpiredAddressIDs(ctx)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		s.logger.DebugContext(ctx, "no expired addresses detected")
		return nil
	}
	s.logger.InfoContext(ctx, "expired addresses detected", slog.Int(AttrKeyCount, len(ids)))

	if err := s.addressDisabler.DisableAddresses(ctx, ids, device.EventSourceExpiry); err != nil {
		return err
	}
	return nil
}
