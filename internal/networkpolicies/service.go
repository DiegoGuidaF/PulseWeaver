package networkpolicies

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type repository interface {
	CreatePolicy(ctx context.Context, p NetworkPolicy) (NetworkPolicy, error)
	GetPolicy(ctx context.Context, id ids.NetworkPolicyID) (NetworkPolicy, error)
	UpdatePolicy(ctx context.Context, p NetworkPolicy) (NetworkPolicy, error)
	DeletePolicy(ctx context.Context, id ids.NetworkPolicyID) error
	SetHostAccess(ctx context.Context, id ids.NetworkPolicyID, bypassHostCheck bool, groupIDs []ids.HostGroupID) error
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

	return p, nil
}

func (s *Service) UpdatePolicy(ctx context.Context, id ids.NetworkPolicyID, fields UpdateFields) (NetworkPolicy, error) {
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
		updatedPolicy = p
		return nil
	})
	if err != nil {
		return NetworkPolicy{}, err
	}

	s.notifyObservers(ctx)
	return updatedPolicy, nil
}

func (s *Service) DeletePolicy(ctx context.Context, id ids.NetworkPolicyID) error {
	if err := s.repo.DeletePolicy(ctx, id); err != nil {
		return err
	}
	s.notifyObservers(ctx)
	return nil
}

func (s *Service) SetHostAccess(ctx context.Context, id ids.NetworkPolicyID, bypassHostCheck bool, groupIDs []ids.HostGroupID) error {
	if err := s.repo.SetHostAccess(ctx, id, bypassHostCheck, groupIDs); err != nil {
		return err
	}

	s.notifyObservers(ctx)
	return nil
}
