package useraccess

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/slicex"
)

type repository interface {
	SetUserAccess(ctx context.Context, userID ids.UserID, bypassHostCheck bool, groupIDs []ids.HostGroupID) error
	EnsureUserSettings(ctx context.Context, userID ids.UserID) error
	DeleteUserData(ctx context.Context, userID ids.UserID) error
	GetAllUserHostSettings(ctx context.Context) ([]UserHostSetting, error)
	GetAllUserHostGrants(ctx context.Context) ([]UserHostGrant, error)
}

type transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	repo      repository
	tx        transactor
	logger    *slog.Logger
	observers []Observer
}

func NewService(repo repository, tx transactor, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		tx:     tx,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "useraccess")),
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
	var (
		settings []UserHostSetting
		grants   []UserHostGrant
	)

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if settings, err = s.repo.GetAllUserHostSettings(ctx); err != nil {
			return err
		}
		if grants, err = s.repo.GetAllUserHostGrants(ctx); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get all user host access: %w", err)
	}

	return mergeUserHostAccess(settings, grants), nil
}

func mergeUserHostAccess(settings []UserHostSetting, groupHosts []UserHostGrant) []policy.UserHostAccess {
	type entry struct {
		bypass bool
		hosts  map[string]struct{}
	}

	byUser := make(map[ids.UserID]*entry, len(settings))
	for _, s := range settings {
		byUser[s.UserID] = &entry{bypass: s.BypassHostCheck}
	}

	for _, g := range groupHosts {
		e := byUser[g.UserID]
		if e == nil {
			continue
		}
		if e.hosts == nil {
			e.hosts = make(map[string]struct{})
		}
		e.hosts[g.FQDN] = struct{}{}
	}

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

func (s *Service) SetUserAccess(ctx context.Context, userID ids.UserID, bypassHostCheck bool, groupIDs []ids.HostGroupID) error {
	groupIDs = slicex.Dedup(groupIDs)
	if err := s.repo.SetUserAccess(ctx, userID, bypassHostCheck, groupIDs); err != nil {
		return err
	}
	s.notifyObservers(ctx)
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
		s.notifyObservers(ctx)
	case auth.EventTypeUserDeleted:
		if err := s.repo.DeleteUserData(ctx, event.UserID); err != nil {
			s.logger.ErrorContext(ctx, "failed to delete user host data",
				slog.Int64("user_id", event.UserID.Int64()),
				slog.Any(logging.AttrKeyError, err),
			)
		}
		s.notifyObservers(ctx)
	}
}
