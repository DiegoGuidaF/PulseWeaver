package hostaccess

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

type repository interface {
	BulkCreateKnownHosts(ctx context.Context, fqdns []string) ([]KnownHost, error)
	UpdateKnownHost(ctx context.Context, id KnownHostID, icon *string) (KnownHost, error)
	DeleteKnownHost(ctx context.Context, id KnownHostID) error

	CreateHostGroupWithMembers(ctx context.Context, name string, description *string, icon *string, hostIDs []KnownHostID) (HostGroupID, error)
	UpdateHostGroupWithMembers(ctx context.Context, id HostGroupID, name string, description *string, icon *string, hostIDs []KnownHostID) error
	UpdateHostGroupMetadata(ctx context.Context, id HostGroupID, name string, description *string, icon *string) error
	DeleteHostGroup(ctx context.Context, id HostGroupID) error

	SetFullUserGrants(ctx context.Context, userID auth.UserID, bypass *bool, hostIDs []KnownHostID, groupIDs []HostGroupID) error

	AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error)
	RemoveIgnoredSuggestionByFQDN(ctx context.Context, fqdn string) error

	EnsureUserSettings(ctx context.Context, userID auth.UserID) error
	DeleteUserData(ctx context.Context, userID auth.UserID) error

	GetAllUserHostAccess(ctx context.Context) ([]policy.UserHostAccess, error)
}

type Service struct {
	repo      repository
	logger    *slog.Logger
	observers []Observer
}

func NewService(repo repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "hostaccess")),
	}
}

func (s *Service) AddObserver(o Observer) {
	if o != nil {
		s.observers = append(s.observers, o)
	}
}

func (s *Service) notifyObservers(ctx context.Context) {
	for _, o := range s.observers {
		o.OnHostAccessChanged(ctx)
	}
}

// GetAllUserHostAccess implements the policy.HostAccessProvider interface.
func (s *Service) GetAllUserHostAccess(ctx context.Context) ([]policy.UserHostAccess, error) {
	return s.repo.GetAllUserHostAccess(ctx)
}

// ── Known hosts ───────────────────────────────────────────────────────────────

func (s *Service) BulkCreateKnownHosts(ctx context.Context, fqdns []string) ([]KnownHost, error) {
	params, err := NewBulkCreateKnownHostsParams(fqdns)
	if err != nil {
		return nil, err
	}
	hosts, err := s.repo.BulkCreateKnownHosts(ctx, params.FQDNs)
	if err != nil {
		return nil, err
	}
	s.notifyObservers(ctx)
	return hosts, nil
}

func (s *Service) UpdateKnownHost(ctx context.Context, id KnownHostID, icon *string) (KnownHost, error) {
	return s.repo.UpdateKnownHost(ctx, id, icon)
}

func (s *Service) DeleteKnownHost(ctx context.Context, id KnownHostID) error {
	if err := s.repo.DeleteKnownHost(ctx, id); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

// ── Host groups ───────────────────────────────────────────────────────────────

func (s *Service) CreateHostGroup(ctx context.Context, name string, description *string, icon *string, hostIDs []KnownHostID) (HostGroupID, error) {
	hostIDs = deduplicateHostIDs(hostIDs)
	groupID, err := s.repo.CreateHostGroupWithMembers(ctx, name, description, icon, hostIDs)
	if err != nil {
		return 0, err
	}
	s.notifyObservers(ctx)
	return groupID, nil
}

// UpdateHostGroup updates a host group's metadata and optionally its members.
// hostIDs semantics: nil = leave members unchanged; non-nil (even empty) = replace members.
func (s *Service) UpdateHostGroup(ctx context.Context, id HostGroupID, name string, description *string, icon *string, hostIDs *[]KnownHostID) error {
	if hostIDs != nil {
		deduped := deduplicateHostIDs(*hostIDs)
		if err := s.repo.UpdateHostGroupWithMembers(ctx, id, name, description, icon, deduped); err != nil {
			return err
		}
		s.notifyObservers(ctx)
		return nil
	}
	return s.repo.UpdateHostGroupMetadata(ctx, id, name, description, icon)
}

func (s *Service) DeleteHostGroup(ctx context.Context, id HostGroupID) error {
	if err := s.repo.DeleteHostGroup(ctx, id); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

// ── User grants ───────────────────────────────────────────────────────────────

func (s *Service) SetFullUserGrants(ctx context.Context, userID auth.UserID, bypass *bool, hostIDs []KnownHostID, groupIDs []HostGroupID) error {
	hostIDs = deduplicateHostIDs(hostIDs)
	groupIDs = deduplicateGroupIDs(groupIDs)
	if err := s.repo.SetFullUserGrants(ctx, userID, bypass, hostIDs, groupIDs); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

// ── Ignored suggestions ───────────────────────────────────────────────────────

func (s *Service) AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error) {
	return s.repo.AddIgnoredSuggestion(ctx, fqdn)
}

func (s *Service) RemoveIgnoredSuggestionByFQDN(ctx context.Context, fqdn string) error {
	return s.repo.RemoveIgnoredSuggestionByFQDN(ctx, fqdn)
}

// ── User lifecycle ────────────────────────────────────────────────────────────

// OnUserEvent implements auth.UserObserver. Called synchronously within the auth
// transaction, so settings changes are atomic with the user lifecycle event.
func (s *Service) OnUserEvent(ctx context.Context, event auth.UserEvent) {
	switch event.Type {
	case auth.EventTypeUserCreated:
		if err := s.repo.EnsureUserSettings(ctx, event.UserID); err != nil {
			s.logger.ErrorContext(ctx, "failed to initialize user host settings",
				slog.Int64("user_id", event.UserID.Int64()),
				slog.Any(logging.AttrKeyError, err),
			)
		}
	case auth.EventTypeUserDeleted:
		if err := s.repo.DeleteUserData(ctx, event.UserID); err != nil {
			s.logger.ErrorContext(ctx, "failed to delete user host data",
				slog.Int64("user_id", event.UserID.Int64()),
				slog.Any(logging.AttrKeyError, err),
			)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func deduplicateHostIDs(ids []KnownHostID) []KnownHostID {
	if ids == nil {
		return nil
	}
	seen := make(map[KnownHostID]struct{}, len(ids))
	out := make([]KnownHostID, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func deduplicateGroupIDs(ids []HostGroupID) []HostGroupID {
	if ids == nil {
		return nil
	}
	seen := make(map[HostGroupID]struct{}, len(ids))
	out := make([]HostGroupID, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
