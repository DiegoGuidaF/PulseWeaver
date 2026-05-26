package devicepairing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// repository is the interface the Service requires from its data layer.
type repository interface {
	CreatePairing(ctx context.Context, p CreatePairingRequest) (*DevicePairing, error)
	GetPairing(ctx context.Context, id ids.DevicePairingID) (*DevicePairing, error)
	ListPairings(ctx context.Context, filter PairingFilter) ([]DevicePairing, error)
	ReplacePendingPairings(ctx context.Context, deviceID ids.DeviceID) error
	InvalidatePairing(ctx context.Context, deviceID ids.DeviceID, id ids.DevicePairingID) error
	ClaimPairing(ctx context.Context, id ids.DevicePairingID) (*DevicePairing, error)
	GetPairingByCode(ctx context.Context, code string) (*DevicePairing, error)
}

type transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// apiKeyManager regenerates the API key for an existing device.
type apiKeyManager interface {
	RegenerateAPIKey(ctx context.Context, deviceID ids.DeviceID) (*device.Device, string, error)
}

// Service contains all business logic for the devicepairing package.
type Service struct {
	repo       repository
	tx         transactor
	keyManager apiKeyManager
	logger     *slog.Logger
}

func NewService(repo repository, tx transactor, keyManager apiKeyManager, logger *slog.Logger) *Service {
	return &Service{
		repo:       repo,
		tx:         tx,
		keyManager: keyManager,
		logger:     logger.With(slog.String(logging.AttrKeyComponent, "devicepairing")),
	}
}

// CreatePairing generates a pairing code and persists the pairing record.
// Any existing pending pairings for the same device are replaced atomically.
func (s *Service) CreatePairing(ctx context.Context, req CreatePairingRequest) (*DevicePairing, error) {
	if err := req.addPairingCode(); err != nil {
		return nil, err
	}

	var pairing *DevicePairing
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.ReplacePendingPairings(ctx, req.DeviceID); err != nil {
			return err
		}
		var err error
		pairing, err = s.repo.CreatePairing(ctx, req)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("persist pairing: %w", err)
	}

	s.logger.InfoContext(ctx, "device pairing created",
		slog.Int64("id", pairing.ID.Int64()),
		slog.Int64("device_id", pairing.DeviceID.Int64()),
		slog.Time("expires_at", pairing.ExpiresAt),
	)
	return pairing, nil
}

// GetPairing returns a single pairing by ID.
func (s *Service) GetPairing(ctx context.Context, id ids.DevicePairingID) (*DevicePairing, error) {
	return s.repo.GetPairing(ctx, id)
}

// ListPairings returns pairings according to the given filter.
func (s *Service) ListPairings(ctx context.Context, filter PairingFilter) ([]DevicePairing, error) {
	return s.repo.ListPairings(ctx, filter)
}

// ClaimPairing validates and redeems a pairing code. On success it returns the
// configuration payload and the plaintext device API key (one-time only).
func (s *Service) ClaimPairing(ctx context.Context, code string) (*ClaimResult, error) {
	claimedPairing := new(DevicePairing)
	var rawAPIKey string

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		pairing, err := s.repo.GetPairingByCode(ctx, code)
		if err != nil {
			return err
		}
		if pairing.Status != StatusPending {
			s.logger.InfoContext(ctx, "attempted to claim non-pending pairing",
				slog.Int64("id", pairing.ID.Int64()),
				slog.String("status", string(pairing.Status)),
			)
			return ErrPairingNotClaimable
		}
		if !pairing.ExpiresAt.After(time.Now().UTC()) {
			s.logger.InfoContext(ctx, "attempted to claim expired pairing", slog.Int64("id", pairing.ID.Int64()))
			return ErrPairingExpired
		}

		_, rawAPIKey, err = s.keyManager.RegenerateAPIKey(ctx, pairing.DeviceID)
		if err != nil {
			return fmt.Errorf("regenerate api key on pairing claim: %w", err)
		}

		claimedPairing, err = s.repo.ClaimPairing(ctx, pairing.ID)
		if err != nil {
			return fmt.Errorf("mark pairing as claimed: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "device pairing claimed")

	result := claimedPairing.ToClaimResult(rawAPIKey)
	return &result, nil
}

// InvalidatePairing soft-deletes an unclaimed pairing.
func (s *Service) InvalidatePairing(ctx context.Context, deviceID ids.DeviceID, id ids.DevicePairingID) error {
	if err := s.repo.InvalidatePairing(ctx, deviceID, id); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "device pairing invalidated", slog.Int64("id", id.Int64()))
	return nil
}
