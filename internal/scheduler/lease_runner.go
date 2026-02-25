package scheduler

import (
	"context"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

func (s *Service) executeAutoExpiry(ctx context.Context) error {
	logger := logging.FromCtx(ctx)
	logger.Info("starting auto-expiry task")

	ids, err := s.expiredAddressFinder.GetExpiredAddressIDs(ctx)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		logger.Debug("no expired addresses detected")
		return nil
	}
	logger.Info("expired addresses detected", slog.Int(AttrKeyCount, len(ids)))

	if err := s.addressDisabler.DisableAddresses(ctx, ids, device.StatusSourceExpiry); err != nil {
		return err
	}
	return nil
}
