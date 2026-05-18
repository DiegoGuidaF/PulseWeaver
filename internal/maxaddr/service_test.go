//go:build test

package maxaddr

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockMaxAddressesProvider struct {
	maxAddresses *int
	err          error
}

func (m *mockMaxAddressesProvider) GetMaxActiveAddresses(_ context.Context, _ ids.DeviceID) (*int, error) {
	return m.maxAddresses, m.err
}

type mockEnabledAddressFetcher struct {
	addresses []device.Address
	err       error
}

func (m *mockEnabledAddressFetcher) GetEnabledAddressesForDevice(_ context.Context, _ ids.DeviceID) ([]device.Address, error) {
	return m.addresses, m.err
}

type mockAddressDisabler struct {
	disabledIDs []ids.AddressID
	err         error
}

func (m *mockAddressDisabler) DisableAddresses(_ context.Context, addressIDs []ids.AddressID, _ device.EventSource) error {
	m.disabledIDs = append(m.disabledIDs, addressIDs...)
	return m.err
}

func newTestService(provider MaxAddressesProvider, fetcher EnabledAddressFetcher, disabler AddressDisabler) *Service {
	return NewService(provider, fetcher, disabler, slog.New(slog.DiscardHandler))
}

// ---------------------------------------------------------------------------
// enforce tests
// ---------------------------------------------------------------------------

func TestService_Enforce_NoRule_NoEviction(t *testing.T) {
	is := is.New(t)
	disabler := &mockAddressDisabler{}
	svc := newTestService(
		&mockMaxAddressesProvider{maxAddresses: nil},
		&mockEnabledAddressFetcher{},
		disabler,
	)

	svc.enforce(context.Background(), ids.DeviceID(1), ids.AddressID(1))

	is.Equal(len(disabler.disabledIDs), 0)
}

func TestService_Enforce_UnderLimit_NoEviction(t *testing.T) {
	is := is.New(t)
	disabler := &mockAddressDisabler{}
	svc := newTestService(
		&mockMaxAddressesProvider{maxAddresses: new(2)},
		&mockEnabledAddressFetcher{
			addresses: []device.Address{
				{ID: ids.AddressID(1), IsEnabled: true},
			},
		},
		disabler,
	)

	svc.enforce(context.Background(), ids.DeviceID(1), ids.AddressID(1))

	is.Equal(len(disabler.disabledIDs), 0)
}

func TestService_Enforce_AtLimit_NoEviction(t *testing.T) {
	is := is.New(t)
	now := time.Now().UTC()
	disabler := &mockAddressDisabler{}
	svc := newTestService(
		&mockMaxAddressesProvider{maxAddresses: new(2)},
		&mockEnabledAddressFetcher{
			addresses: []device.Address{
				{ID: ids.AddressID(2), IsEnabled: true, UpdatedAt: now},
				{ID: ids.AddressID(1), IsEnabled: true, UpdatedAt: now.Add(-time.Minute)},
			},
		},
		disabler,
	)

	svc.enforce(context.Background(), ids.DeviceID(1), ids.AddressID(2))

	is.Equal(len(disabler.disabledIDs), 0)
}

func TestService_Enforce_OverLimit_EvictsOldest(t *testing.T) {
	is := is.New(t)
	now := time.Now().UTC()
	disabler := &mockAddressDisabler{}

	// 3 addresses returned newest-first (DESC)
	addr1 := device.Address{ID: ids.AddressID(1), IsEnabled: true, UpdatedAt: now.Add(-2 * time.Minute)}
	addr2 := device.Address{ID: ids.AddressID(2), IsEnabled: true, UpdatedAt: now.Add(-time.Minute)}
	addr3 := device.Address{ID: ids.AddressID(3), IsEnabled: true, UpdatedAt: now}

	svc := newTestService(
		&mockMaxAddressesProvider{maxAddresses: new(2)},
		&mockEnabledAddressFetcher{
			addresses: []device.Address{addr3, addr2, addr1}, // DESC order
		},
		disabler,
	)

	svc.enforce(context.Background(), ids.DeviceID(1), ids.AddressID(3))

	is.Equal(len(disabler.disabledIDs), 1)
	is.Equal(disabler.disabledIDs[0], ids.AddressID(1)) // oldest evicted
}

func TestService_Enforce_NewAddressNotEvicted(t *testing.T) {
	is := is.New(t)
	now := time.Now().UTC()
	disabler := &mockAddressDisabler{}

	// just-registered address is the oldest — it must be protected
	addr1 := device.Address{ID: ids.AddressID(1), IsEnabled: true, UpdatedAt: now.Add(-2 * time.Minute)}
	addr2 := device.Address{ID: ids.AddressID(2), IsEnabled: true, UpdatedAt: now.Add(-time.Minute)}
	addr3 := device.Address{ID: ids.AddressID(3), IsEnabled: true, UpdatedAt: now}

	svc := newTestService(
		&mockMaxAddressesProvider{maxAddresses: new(2)},
		&mockEnabledAddressFetcher{
			addresses: []device.Address{addr3, addr2, addr1}, // DESC order
		},
		disabler,
	)

	// justRegisteredID is addr1 (the oldest) — should be protected; addr2 evicted instead
	svc.enforce(context.Background(), ids.DeviceID(1), ids.AddressID(1))

	is.Equal(len(disabler.disabledIDs), 1)
	is.Equal(disabler.disabledIDs[0], ids.AddressID(2)) // next-oldest evicted, not addr1
}

func TestService_Enforce_ProviderError_BestEffort(t *testing.T) {
	is := is.New(t)
	disabler := &mockAddressDisabler{}
	svc := newTestService(
		&mockMaxAddressesProvider{err: errors.New("db error")},
		&mockEnabledAddressFetcher{},
		disabler,
	)

	// Must not panic; disabler must not be called
	svc.enforce(context.Background(), ids.DeviceID(1), ids.AddressID(1))

	is.Equal(len(disabler.disabledIDs), 0)
}

func TestService_Enforce_FetcherError_BestEffort(t *testing.T) {
	is := is.New(t)
	disabler := &mockAddressDisabler{}
	svc := newTestService(
		&mockMaxAddressesProvider{maxAddresses: new(2)},
		&mockEnabledAddressFetcher{err: errors.New("db error")},
		disabler,
	)

	// Must not panic; disabler must not be called
	svc.enforce(context.Background(), ids.DeviceID(1), ids.AddressID(1))

	is.Equal(len(disabler.disabledIDs), 0)
}

// ---------------------------------------------------------------------------
// OnAddressEvent tests
// ---------------------------------------------------------------------------

func TestService_OnAddressEvent_FiltersRefreshed(t *testing.T) {
	is := is.New(t)
	svc := newTestService(&mockMaxAddressesProvider{}, &mockEnabledAddressFetcher{}, &mockAddressDisabler{})

	svc.OnAddressEvent(context.Background(), device.AddressEvent{
		Type:      device.EventTypeAddressRefreshed,
		DeviceID:  ids.DeviceID(1),
		AddressID: ids.AddressID(1),
	})

	is.Equal(len(svc.events), 0)
}

func TestService_OnAddressEvent_FiltersDisabled(t *testing.T) {
	is := is.New(t)
	svc := newTestService(&mockMaxAddressesProvider{}, &mockEnabledAddressFetcher{}, &mockAddressDisabler{})

	svc.OnAddressEvent(context.Background(), device.AddressEvent{
		Type:      device.EventTypeAddressDisabled,
		DeviceID:  ids.DeviceID(1),
		AddressID: ids.AddressID(1),
	})

	is.Equal(len(svc.events), 0)
}

func TestService_OnAddressEvent_PassesCreated(t *testing.T) {
	is := is.New(t)
	svc := newTestService(&mockMaxAddressesProvider{}, &mockEnabledAddressFetcher{}, &mockAddressDisabler{})

	svc.OnAddressEvent(context.Background(), device.AddressEvent{
		Type:      device.EventTypeAddressCreated,
		DeviceID:  ids.DeviceID(1),
		AddressID: ids.AddressID(1),
	})

	is.Equal(len(svc.events), 1)
}

func TestService_OnAddressEvent_PassesEnabled(t *testing.T) {
	is := is.New(t)
	svc := newTestService(&mockMaxAddressesProvider{}, &mockEnabledAddressFetcher{}, &mockAddressDisabler{})

	svc.OnAddressEvent(context.Background(), device.AddressEvent{
		Type:      device.EventTypeAddressEnabled,
		DeviceID:  ids.DeviceID(1),
		AddressID: ids.AddressID(1),
	})

	is.Equal(len(svc.events), 1)
}

// ---------------------------------------------------------------------------
// RunListener tests
// ---------------------------------------------------------------------------

func TestService_RunListener_ContextCancel_ExitsCleanly(t *testing.T) {
	is := is.New(t)
	svc := newTestService(
		&mockMaxAddressesProvider{maxAddresses: nil},
		&mockEnabledAddressFetcher{},
		&mockAddressDisabler{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- svc.RunListener(ctx)
	}()

	cancel()

	err := <-done
	is.NoErr(err)
}
