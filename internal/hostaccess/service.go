package hostaccess

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

type repository interface {
	ListKnownHosts(ctx context.Context) ([]KnownHost, error)
	CreateKnownHost(ctx context.Context, draft KnownHostDraft) (KnownHostID, error)
	UpdateKnownHost(ctx context.Context, id KnownHostID, icon *string) (KnownHost, error)
	DeleteKnownHost(ctx context.Context, id KnownHostID) error
	ListKnownHostsByIDs(ctx context.Context, ids []KnownHostID) ([]KnownHost, error)
	SetKnownHostGroupMembership(ctx context.Context, hostID KnownHostID, groupIDs []HostGroupID) error

	ListHostGroups(ctx context.Context) ([]HostGroup, error)
	CreateHostGroup(ctx context.Context, draft HostGroupDraft) (HostGroupID, error)
	UpdateHostGroup(ctx context.Context, group HostGroup) error
	DeleteHostGroup(ctx context.Context, id HostGroupID) error

	SetFullUserGrants(ctx context.Context, userID auth.UserID, bypass *bool, hostIDs []KnownHostID, groupIDs []HostGroupID) error

	AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error)
	RemoveIgnoredSuggestionByFQDN(ctx context.Context, fqdn string) error

	EnsureUserSettings(ctx context.Context, userID auth.UserID) error
	DeleteUserData(ctx context.Context, userID auth.UserID) error

	GetAllUserHostSettings(ctx context.Context) ([]UserHostSetting, error)
	GetAllUserDirectHostGrants(ctx context.Context) ([]UserHostGrant, error)
	GetAllUserGroupHostGrants(ctx context.Context) ([]UserHostGrant, error)
}

type transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	repo                    repository
	tx                      transactor
	logger                  *slog.Logger
	userHostAccessObservers []Observer
}

func NewService(repo repository, tx transactor, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		tx:     tx,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "hostaccess")),
	}
}

func (s *Service) AddUserHostAccessObserver(o Observer) {
	if o != nil {
		s.userHostAccessObservers = append(s.userHostAccessObservers, o)
	}
}

func (s *Service) notifyUserHostAccessObservers(ctx context.Context) {
	for _, o := range s.userHostAccessObservers {
		o.OnHostAccessChanged(ctx)
	}
}

// GetAllUserHostAccess implements the policy.HostAccessProvider interface.
func (s *Service) GetAllUserHostAccess(ctx context.Context) ([]policy.UserHostAccess, error) {
	var (
		settings    []UserHostSetting
		directHosts []UserHostGrant
		groupHosts  []UserHostGrant
	)

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if settings, err = s.repo.GetAllUserHostSettings(ctx); err != nil {
			return err
		}
		if directHosts, err = s.repo.GetAllUserDirectHostGrants(ctx); err != nil {
			return err
		}
		if groupHosts, err = s.repo.GetAllUserGroupHostGrants(ctx); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get all user host access: %w", err)
	}

	return mergeUserHostAccess(settings, directHosts, groupHosts), nil
}

func mergeUserHostAccess(settings []UserHostSetting, directHosts, groupHosts []UserHostGrant) []policy.UserHostAccess {
	type entry struct {
		bypass bool
		hosts  map[string]struct{}
	}

	byUser := make(map[auth.UserID]*entry, len(settings))
	for _, s := range settings {
		byUser[s.UserID] = &entry{bypass: s.BypassAllowlist}
	}

	addGrants := func(grants []UserHostGrant) {
		for _, g := range grants {
			e := byUser[g.UserID]
			if e == nil {
				continue
			}
			if e.hosts == nil {
				e.hosts = make(map[string]struct{})
			}
			e.hosts[g.FQDN] = struct{}{}
		}
	}
	addGrants(directHosts)
	addGrants(groupHosts)

	result := make([]policy.UserHostAccess, 0, len(byUser))
	for userID, e := range byUser {
		if !e.bypass && len(e.hosts) == 0 {
			continue
		}
		hosts := make([]string, 0, len(e.hosts))
		for h := range e.hosts {
			hosts = append(hosts, h)
		}
		result = append(result, policy.UserHostAccess{
			UserID:          userID,
			BypassAllowlist: e.bypass,
			AllowedHosts:    hosts,
		})
	}
	return result
}

func (s *Service) ListHostGroups(ctx context.Context) ([]HostGroup, error) {
	return s.repo.ListHostGroups(ctx)
}

func (s *Service) AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error) {
	normalised := NormaliseFQDN(fqdn)
	if err := ValidateFQDN(normalised); err != nil {
		return IgnoredHostSuggestion{}, err
	}
	return s.repo.AddIgnoredSuggestion(ctx, normalised)
}

func (s *Service) RemoveIgnoredSuggestionByFQDN(ctx context.Context, fqdn string) error {
	return s.repo.RemoveIgnoredSuggestionByFQDN(ctx, fqdn)
}

func (s *Service) SetFullUserGrants(ctx context.Context, userID auth.UserID, bypass *bool, hostIDs []KnownHostID, groupIDs []HostGroupID) error {
	hostIDs = deduplicateHostIDs(hostIDs)
	groupIDs = deduplicateGroupIDs(groupIDs)
	if err := s.repo.SetFullUserGrants(ctx, userID, bypass, hostIDs, groupIDs); err != nil {
		return err
	}
	s.notifyUserHostAccessObservers(ctx)
	return nil
}

func (s *Service) OnUserEvent(ctx context.Context, event auth.UserEvent) {
	switch event.Type {
	case auth.EventTypeUserCreated:
		if err := s.repo.EnsureUserSettings(ctx, event.UserID); err != nil {
			s.logger.ErrorContext(ctx, "failed to initialize user host settings",
				slog.Int64("user_id", event.UserID.Int64()),
				slog.Any(logging.AttrKeyError, err),
			)
		}
		s.notifyUserHostAccessObservers(ctx)
	case auth.EventTypeUserDeleted:
		if err := s.repo.DeleteUserData(ctx, event.UserID); err != nil {
			s.logger.ErrorContext(ctx, "failed to delete user host data",
				slog.Int64("user_id", event.UserID.Int64()),
				slog.Any(logging.AttrKeyError, err),
			)
		}
		s.notifyUserHostAccessObservers(ctx)
	}
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
