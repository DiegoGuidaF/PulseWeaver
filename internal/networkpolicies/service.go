package networkpolicies

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type repository interface {
	CreatePolicy(ctx context.Context, p NetworkPolicy) (NetworkPolicy, error)
	GetPolicy(ctx context.Context, id NetworkPolicyID) (*NetworkPolicy, error)
	UpdatePolicy(ctx context.Context, p NetworkPolicy) (*NetworkPolicy, error)
	DeletePolicy(ctx context.Context, id NetworkPolicyID) error
	SetHostAccess(ctx context.Context, id NetworkPolicyID, allowAll bool, groupIDs []int64, hostIDs []int64) error
}

type transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	repo      repository
	tx        transactor
	logger    *slog.Logger
	observers []PolicyChangeObserver
}

func NewService(repo repository, tx transactor, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		tx:     tx,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "networkpolicies")),
	}
}

func (s *Service) AddObserver(o PolicyChangeObserver) {
	s.observers = append(s.observers, o)
}

func (s *Service) notifyObservers(ctx context.Context) {
	for _, obs := range s.observers {
		obs.OnNetworkPolicyChanged(ctx)
	}
}

func (s *Service) CreatePolicy(ctx context.Context, name, cidr string, description *string) (NetworkPolicy, error) {
	normalized, err := normalizeCIDR(cidr)
	if err != nil {
		return NetworkPolicy{}, fmt.Errorf("%w: %s", ErrInvalidCIDR, cidr)
	}

	p, err := s.repo.CreatePolicy(ctx, NetworkPolicy{
		Name:        name,
		CIDR:        normalized,
		Description: description,
		Enabled:     true,
	})
	if err != nil {
		return NetworkPolicy{}, err
	}

	// TODO: Since when creating a policy there are no configured hosts, it won't affect the policy cache, I don't
	// believe we should notify here
	//s.notifyObservers(ctx)
	return p, nil
}

func (s *Service) UpdatePolicy(ctx context.Context, id NetworkPolicyID, fields UpdateFields) (NetworkPolicy, error) {
	updatedPolicy := NetworkPolicy{}
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		existing, err := s.repo.GetPolicy(ctx, id)
		if err != nil {
			return err
		}

		updated, err := existing.Apply(fields)
		if err != nil {
			return err
		}

		p, err := s.repo.UpdatePolicy(ctx, updated)
		if err != nil {
			return err
		}
		updatedPolicy = *p
		return nil
	})
	if err != nil {
		return NetworkPolicy{}, err
	}

	//TODO: Maybe we should only notify if the "enabled" changed?
	s.notifyObservers(ctx)
	return updatedPolicy, nil
}

func (s *Service) DeletePolicy(ctx context.Context, id NetworkPolicyID) error {
	if err := s.repo.DeletePolicy(ctx, id); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

func (s *Service) SetHostAccess(ctx context.Context, id NetworkPolicyID, allowAll bool, groupIDs, hostIDs []int64) error {
	actualGroupIDs := groupIDs
	actualHostIDs := hostIDs
	//TODO: We shouldn't remove the hostIds and groups, they should be ignored when set in the cache since the allow_all already sets all hosts
	// However if a user sets that and then removes it, the previous groups and hosts should remain
	// Check that the same happens currently for the hosts
	if allowAll {
		actualGroupIDs = nil
		actualHostIDs = nil
	}

	if err := s.repo.SetHostAccess(ctx, id, allowAll, actualGroupIDs, actualHostIDs); err != nil {
		return err
	}

	s.notifyObservers(ctx)
	return nil
}
