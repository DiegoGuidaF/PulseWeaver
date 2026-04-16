package registration

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// repository is the interface the Service requires from its data layer.
type repository interface {
	CreateInvite(ctx context.Context, p *PendingRegistration) error
	GetInvite(ctx context.Context, id string) (*PendingRegistration, error)
	ListInvites(ctx context.Context, filter InviteFilter) ([]*PendingRegistration, error)
	InvalidateInvite(ctx context.Context, id string) error
	ClaimInvite(ctx context.Context, code string) (*ClaimResult, error)
}

// Service contains all business logic for the registration package.
type Service struct {
	repo   repository
	logger *slog.Logger
}

func NewService(repo repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "registration")),
	}
}

// CreateInvite generates a registration code and persists the invite.
// The device API key is generated later at claim time — not pre-staged here.
func (s *Service) CreateInvite(ctx context.Context, req CreateInviteRequest) (*PendingRegistration, error) {
	code, _, err := generateRegistrationCode(req.HeartbeatServerURL)
	if err != nil {
		return nil, fmt.Errorf("generate registration code: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(req.ExpiresInHours) * time.Hour)

	p := &PendingRegistration{
		ID:                  generateID(),
		DeviceName:          req.DeviceName,
		OwnerID:             req.OwnerID,
		RegistrationCode:    &code,
		HeartbeatServerURL:  req.HeartbeatServerURL,
		IntervalSeconds:     req.IntervalSeconds,
		AppBiometricEnabled: req.AppBiometricEnabled,
		AppSettingsLocked:   req.AppSettingsLocked,
		ExpiresAt:           expiresAt,
		CreatedAt:           now,
	}

	if err := s.repo.CreateInvite(ctx, p); err != nil {
		return nil, fmt.Errorf("persist invite: %w", err)
	}

	s.logger.InfoContext(ctx, "registration invite created",
		slog.String("id", p.ID),
		slog.String("device_name", p.DeviceName),
		slog.Time("expires_at", p.ExpiresAt),
	)
	return p, nil
}

// GetInvite returns a single invite by ID.
func (s *Service) GetInvite(ctx context.Context, id string) (*PendingRegistration, error) {
	p, err := s.repo.GetInvite(ctx, id)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListInvites returns invites according to the given filter.
func (s *Service) ListInvites(ctx context.Context, filter InviteFilter) ([]*PendingRegistration, error) {
	return s.repo.ListInvites(ctx, filter)
}

// ClaimInvite validates and redeems a registration code. On success it returns the
// configuration payload and the plaintext device API key (one-time only).
func (s *Service) ClaimInvite(ctx context.Context, code string) (*ClaimResult, error) {
	result, err := s.repo.ClaimInvite(ctx, code)
	if err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "registration invite claimed")
	return result, nil
}

// InvalidateInvite hard-deletes an unclaimed invite.
func (s *Service) InvalidateInvite(ctx context.Context, id string) error {
	if err := s.repo.InvalidateInvite(ctx, id); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "registration invite invalidated", slog.String("id", id))
	return nil
}
