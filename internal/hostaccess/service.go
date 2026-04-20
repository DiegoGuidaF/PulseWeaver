package hostaccess

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

type repository interface {
	CreateKnownHost(ctx context.Context, fqdn string, icon *string) (KnownHost, error)
	BulkCreateKnownHosts(ctx context.Context, fqdns []string) ([]KnownHost, error)
	GetKnownHost(ctx context.Context, id KnownHostID) (KnownHost, error)
	ListKnownHosts(ctx context.Context) ([]KnownHost, error)
	UpdateKnownHost(ctx context.Context, id KnownHostID, icon *string) (KnownHost, error)
	DeleteKnownHost(ctx context.Context, id KnownHostID) error

	CreateHostGroup(ctx context.Context, name string, description *string, icon *string) (HostGroup, error)
	GetHostGroup(ctx context.Context, id HostGroupID) (HostGroup, error)
	ListHostGroups(ctx context.Context) ([]HostGroup, error)
	ListHostGroupsWithMembers(ctx context.Context) ([]HostGroupWithMembers, error)
	UpdateHostGroup(ctx context.Context, id HostGroupID, name string, description *string, icon *string) (HostGroup, error)
	DeleteHostGroup(ctx context.Context, id HostGroupID) error

	AddHostToGroup(ctx context.Context, groupID HostGroupID, hostID KnownHostID) error
	RemoveHostFromGroup(ctx context.Context, groupID HostGroupID, hostID KnownHostID) error
	ListHostGroupMembers(ctx context.Context, groupID HostGroupID) ([]KnownHost, error)
	SetHostGroupMembers(ctx context.Context, groupID HostGroupID, hostIDs []KnownHostID) error

	GrantUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error
	RevokeUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error
	GrantUserHostGroup(ctx context.Context, userID auth.UserID, groupID HostGroupID) error
	RevokeUserHostGroup(ctx context.Context, userID auth.UserID, groupID HostGroupID) error
	ListUserGrants(ctx context.Context, userID auth.UserID) (hosts []KnownHost, groups []HostGroup, err error)
	SetUserGrants(ctx context.Context, userID auth.UserID, hostIDs []KnownHostID, groupIDs []HostGroupID) error
	SetUserBypassAllowlist(ctx context.Context, userID auth.UserID, bypass bool) error

	AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error)
	FindIgnoredSuggestionByFQDN(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error)
	RemoveIgnoredSuggestion(ctx context.Context, id int64) error
	ListIgnoredSuggestions(ctx context.Context) ([]IgnoredHostSuggestion, error)

	GetUserBypassAllowlist(ctx context.Context, userID auth.UserID) (bool, error)

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

// TODO: Not used, can be removed
func (s *Service) CreateKnownHost(ctx context.Context, fqdn string, icon *string) (KnownHost, error) {
	host, err := s.repo.CreateKnownHost(ctx, fqdn, icon)
	if err != nil {
		return KnownHost{}, err
	}
	s.notifyObservers(ctx)
	return host, nil
}

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

func (s *Service) CreateHostGroup(ctx context.Context, name string, description *string, icon *string) (HostGroup, error) {
	group, err := s.repo.CreateHostGroup(ctx, name, description, icon)
	if err != nil {
		return HostGroup{}, err
	}
	s.notifyObservers(ctx)
	return group, nil
}

func (s *Service) ListHostGroupsWithMembers(ctx context.Context) ([]HostGroupWithMembers, error) {
	return s.repo.ListHostGroupsWithMembers(ctx)
}

func (s *Service) UpdateHostGroup(ctx context.Context, id HostGroupID, name string, description *string, icon *string) (HostGroup, error) {
	group, err := s.repo.UpdateHostGroup(ctx, id, name, description, icon)
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

func (s *Service) SetHostGroupMembers(ctx context.Context, groupID HostGroupID, hostIDs []KnownHostID) error {
	params := NewSetHostGroupMembersParams(groupID, hostIDs)
	if err := s.repo.SetHostGroupMembers(ctx, params.GroupID, params.HostIDs); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
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

func (s *Service) SetUserGrants(ctx context.Context, userID auth.UserID, hostIDs []KnownHostID, groupIDs []HostGroupID) error {
	params := NewSetUserGrantsParams(userID, hostIDs, groupIDs)
	if err := s.repo.SetUserGrants(ctx, params.UserID, params.HostIDs, params.GroupIDs); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

func (s *Service) SetUserBypassAllowlist(ctx context.Context, userID auth.UserID, bypass bool) error {
	if err := s.repo.SetUserBypassAllowlist(ctx, userID, bypass); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

// ── Ignored suggestions ───────────────────────────────────────────────────────

func (s *Service) AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error) {
	return s.repo.AddIgnoredSuggestion(ctx, fqdn)
}

func (s *Service) FindIgnoredSuggestionByFQDN(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error) {
	return s.repo.FindIgnoredSuggestionByFQDN(ctx, fqdn)
}

func (s *Service) GetUserBypassAllowlist(ctx context.Context, userID auth.UserID) (bool, error) {
	return s.repo.GetUserBypassAllowlist(ctx, userID)
}

func (s *Service) RemoveIgnoredSuggestion(ctx context.Context, id int64) error {
	return s.repo.RemoveIgnoredSuggestion(ctx, id)
}

func (s *Service) ListIgnoredSuggestions(ctx context.Context) ([]IgnoredHostSuggestion, error) {
	return s.repo.ListIgnoredSuggestions(ctx)
}
