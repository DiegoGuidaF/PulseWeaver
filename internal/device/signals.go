package device

import (
	"context"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

func (s *Service) signalAddressStateChanged(ctx context.Context) {
	if s.addressStateChanged == nil {
		return
	}
	select {
	case s.addressStateChanged <- struct{}{}:
	default:
		logger := logging.FromCtx(ctx)
		logger.Warn("address state channel channel full, dropping signal")
	}
}

func (s *Service) publishAddressEvent(ctx context.Context, event AddressEvent) {
	if s.events == nil {
		return
	}
	select {
	case s.events <- event:
	default:
		logger := logging.FromCtx(ctx)
		logger.Warn("event channel full, dropped event", slog.Any("event", event))
	}
}
