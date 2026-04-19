package hostaccess

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

type repository interface {
	CreateKnownHost(ctx context.Context, fqdn string) (KnownHost, error)
	GetKnownHost(ctx context.Context, id KnownHostID) (KnownHost, error)
	ListKnownHosts(ctx context.Context) ([]KnownHost, error)
	DeleteKnownHost(ctx context.Context, id KnownHostID) error

	CreateHostGroup(ctx context.Context, name string, description *string) (HostGroup, error)
	GetHostGroup(ctx context.Context, id HostGroupID) (HostGroup, error)
	ListHostGroups(ctx context.Context) ([]HostGroup, error)
	DeleteHostGroup(ctx context.Context, id HostGroupID) error

	AddHostToGroup(ctx context.Context, groupID HostGroupID, hostID KnownHostID) error
	RemoveHostFromGroup(ctx context.Context, groupID HostGroupID, hostID KnownHostID) error
	ListHostGroupMembers(ctx context.Context, groupID HostGroupID) ([]KnownHost, error)

	GrantUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error
	RevokeUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error
	GrantUserHostGroup(ctx context.Context, userID auth.UserID, groupID HostGroupID) error
	RevokeUserHostGroup(ctx context.Context, userID auth.UserID, groupID HostGroupID) error
	ListUserGrants(ctx context.Context, userID auth.UserID) (hosts []KnownHost, groups []HostGroup, err error)

	AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error)
	RemoveIgnoredSuggestion(ctx context.Context, id int64) error
	ListIgnoredSuggestions(ctx context.Context) ([]IgnoredHostSuggestion, error)

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

func (s *Service) CreateKnownHost(ctx context.Context, fqdn string) (KnownHost, error) {
	host, err := s.repo.CreateKnownHost(ctx, fqdn)
	if err != nil {
		return KnownHost{}, err
	}
	s.notifyObservers(ctx)
	return host, nil
}

func (s *Service) GetKnownHost(ctx context.Context, id KnownHostID) (KnownHost, error) {
	return s.repo.GetKnownHost(ctx, id)
}

func (s *Service) ListKnownHosts(ctx context.Context) ([]KnownHost, error) {
	return s.repo.ListKnownHosts(ctx)
}

func (s *Service) DeleteKnownHost(ctx context.Context, id KnownHostID) error {
	if err := s.repo.DeleteKnownHost(ctx, id); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

// ── Host groups ───────────────────────────────────────────────────────────────

func (s *Service) CreateHostGroup(ctx context.Context, name string, description *string) (HostGroup, error) {
	group, err := s.repo.CreateHostGroup(ctx, name, description)
	if err != nil {
		return HostGroup{}, err
	}
	s.notifyObservers(ctx)
	return group, nil
}

func (s *Service) GetHostGroup(ctx context.Context, id HostGroupID) (HostGroup, error) {
	return s.repo.GetHostGroup(ctx, id)
}

func (s *Service) ListHostGroups(ctx context.Context) ([]HostGroup, error) {
	return s.repo.ListHostGroups(ctx)
}

func (s *Service) DeleteHostGroup(ctx context.Context, id HostGroupID) error {
	if err := s.repo.DeleteHostGroup(ctx, id); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

// ── Host group members ────────────────────────────────────────────────────────

func (s *Service) AddHostToGroup(ctx context.Context, groupID HostGroupID, hostID KnownHostID) error {
	if err := s.repo.AddHostToGroup(ctx, groupID, hostID); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

func (s *Service) RemoveHostFromGroup(ctx context.Context, groupID HostGroupID, hostID KnownHostID) error {
	if err := s.repo.RemoveHostFromGroup(ctx, groupID, hostID); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

func (s *Service) ListHostGroupMembers(ctx context.Context, groupID HostGroupID) ([]KnownHost, error) {
	return s.repo.ListHostGroupMembers(ctx, groupID)
}

// ── User grants ───────────────────────────────────────────────────────────────

func (s *Service) GrantUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error {
	if err := s.repo.GrantUserHost(ctx, userID, hostID); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

func (s *Service) RevokeUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error {
	if err := s.repo.RevokeUserHost(ctx, userID, hostID); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

func (s *Service) GrantUserHostGroup(ctx context.Context, userID auth.UserID, groupID HostGroupID) error {
	if err := s.repo.GrantUserHostGroup(ctx, userID, groupID); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

func (s *Service) RevokeUserHostGroup(ctx context.Context, userID auth.UserID, groupID HostGroupID) error {
	if err := s.repo.RevokeUserHostGroup(ctx, userID, groupID); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

func (s *Service) ListUserGrants(ctx context.Context, userID auth.UserID) (hosts []KnownHost, groups []HostGroup, err error) {
	return s.repo.ListUserGrants(ctx, userID)
}

// ── Ignored suggestions ───────────────────────────────────────────────────────

func (s *Service) AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error) {
	return s.repo.AddIgnoredSuggestion(ctx, fqdn)
}

func (s *Service) RemoveIgnoredSuggestion(ctx context.Context, id int64) error {
	return s.repo.RemoveIgnoredSuggestion(ctx, id)
}

func (s *Service) ListIgnoredSuggestions(ctx context.Context) ([]IgnoredHostSuggestion, error) {
	return s.repo.ListIgnoredSuggestions(ctx)
}
