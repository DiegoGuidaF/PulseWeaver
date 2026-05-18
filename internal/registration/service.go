package registration

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// repository is the interface the Service requires from its data layer.
type repository interface {
	CreateInvite(ctx context.Context, p CreateInviteRequest) (*PendingRegistration, error)
	GetInvite(ctx context.Context, id PendingRegistrationID) (*PendingRegistration, error)
	ListInvites(ctx context.Context, filter InviteFilter) ([]PendingRegistration, error)
	InvalidateInvite(ctx context.Context, id PendingRegistrationID) error
	ClaimInvite(ctx context.Context, id PendingRegistrationID, deviceID ids.DeviceID) (*PendingRegistration, error)
	GetInviteByCode(ctx context.Context, code string) (*PendingRegistration, error)
}

type transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// deviceProvisioner Allows creating a device with a given apiKey
type deviceProvisioner interface {
	CreateDeviceWithAPIKey(ctx context.Context, name string, ownerID ids.UserID) (deviceID ids.DeviceID, rawAPIKey string, err error)
}

// Service contains all business logic for the registration package.
type Service struct {
	repo              repository
	tx                transactor
	deviceProvisioner deviceProvisioner
	logger            *slog.Logger
}

func NewService(repo repository, tx transactor, deviceProvisioner deviceProvisioner, logger *slog.Logger) *Service {
	return &Service{
		repo:              repo,
		tx:                tx,
		deviceProvisioner: deviceProvisioner,
		logger:            logger.With(slog.String(logging.AttrKeyComponent, "registration")),
	}
}

// CreateInvite generates a registration code and persists the invite.
// The device API key is generated later at claim time — not pre-staged here.
func (s *Service) CreateInvite(ctx context.Context, req CreateInviteRequest) (*PendingRegistration, error) {
	err := req.addRegistrationCode()
	if err != nil {
		return nil, err
	}

	reg, err := s.repo.CreateInvite(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("persist invite: %w", err)
	}

	s.logger.InfoContext(ctx, "registration invite created",
		slog.Int64("id", reg.ID.Int64()),
		slog.String("device_name", reg.DeviceName),
		slog.Time("expires_at", reg.ExpiresAt),
	)
	return reg, nil
}

// GetInvite returns a single invite by ID.
func (s *Service) GetInvite(ctx context.Context, id PendingRegistrationID) (*PendingRegistration, error) {
	p, err := s.repo.GetInvite(ctx, id)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListInvites returns invites according to the given filter.
func (s *Service) ListInvites(ctx context.Context, filter InviteFilter) ([]PendingRegistration, error) {
	return s.repo.ListInvites(ctx, filter)
}

// ClaimInvite validates and redeems a registration code. On success it returns the
// configuration payload and the plaintext device API key (one-time only).
func (s *Service) ClaimInvite(ctx context.Context, code string) (*ClaimResult, error) {
	claimedInvite := new(PendingRegistration)
	var rawAPIKey string

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		invite, err := s.repo.GetInviteByCode(ctx, code)
		if err != nil {
			return err
		}
		if invite.UsedAt != nil || invite.InvalidatedAt != nil {
			s.logger.InfoContext(ctx, "attempted to claim already used or invalidated invite", slog.Int64("id", invite.ID.Int64()))
			return ErrInviteNotClaimable
		}
		if !invite.ExpiresAt.After(time.Now().UTC()) {
			s.logger.InfoContext(ctx, "attempted to claim expired invite", slog.Int64("id", invite.ID.Int64()))
			return ErrInviteExpired
		}

		var deviceID ids.DeviceID
		deviceID, rawAPIKey, err = s.deviceProvisioner.CreateDeviceWithAPIKey(ctx, invite.DeviceName, invite.OwnerID)
		if err != nil {
			return fmt.Errorf("register device via invitation claim: %w", err)
		}

		claimedInvite, err = s.repo.ClaimInvite(ctx, invite.ID, deviceID)
		if err != nil {
			return fmt.Errorf("mark invite as claimed: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "registration invite claimed")

	result := claimedInvite.ToClaimResult(rawAPIKey)
	return &result, nil
}

// InvalidateInvite hard-deletes an unclaimed invite.
func (s *Service) InvalidateInvite(ctx context.Context, id PendingRegistrationID) error {
	if err := s.repo.InvalidateInvite(ctx, id); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "registration invite invalidated", slog.Int64("id", id.Int64()))
	return nil
}
