package device

import (
	"context"
	"fmt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDevices(ctx context.Context) ([]Device, error) {
	devices, err := s.repo.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("get devices: %w", err)
	}

	return devices, nil
}
