package registration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// repository is the interface the Service requires from its data layer.
type repository interface {
	CreateInvite(ctx context.Context, p CreateInviteRequest) (*PendingRegistration, error)
	GetInvite(ctx context.Context, id PendingRegistrationID) (*PendingRegistration, error)
	ListInvites(ctx context.Context, filter InviteFilter) ([]PendingRegistration, error)
	InvalidateInvite(ctx context.Context, id PendingRegistrationID) error
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
	result, err := s.repo.ClaimInvite(ctx, code)
	if err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "registration invite claimed")
	return result, nil
}

// InvalidateInvite hard-deletes an unclaimed invite.
func (s *Service) InvalidateInvite(ctx context.Context, id PendingRegistrationID) error {
	if err := s.repo.InvalidateInvite(ctx, id); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "registration invite invalidated", slog.Int64("id", id.Int64()))
	return nil
}
