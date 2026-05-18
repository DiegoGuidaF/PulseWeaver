package hosts

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type repository interface {
	ListHosts(ctx context.Context) ([]Host, error)
	CreateHost(ctx context.Context, draft HostDraft) (ids.HostID, error)
	DeleteHost(ctx context.Context, id ids.HostID) error
	ListHostsByIDs(ctx context.Context, ids []ids.HostID) ([]Host, error)
	SetHostGroupMembership(ctx context.Context, hostID ids.HostID, groupIDs []ids.HostGroupID) error

	ListHostGroups(ctx context.Context) ([]HostGroup, error)
	CreateHostGroup(ctx context.Context, draft HostGroupDraft) (ids.HostGroupID, error)
	UpdateHostGroup(ctx context.Context, group HostGroup) error
	DeleteHostGroup(ctx context.Context, id ids.HostGroupID) error

	AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error)
	RemoveIgnoredSuggestionByFQDN(ctx context.Context, fqdn string) error
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
		logger: logger.With(slog.String(logging.AttrKeyComponent, "hosts")),
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
