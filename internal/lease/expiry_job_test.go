//go:build test

package lease

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

type fakeDisabler struct {
	calledWith []ids.AddressID
	source     device.EventSource
	err        error
	calls      int
}

func (f *fakeDisabler) DisableAddresses(_ context.Context, ids []ids.AddressID, source device.EventSource) error {
	f.calls++
	f.calledWith = ids
	f.source = source
	return f.err
}

var _ AddressDisabler = (*fakeDisabler)(nil)

func newExpiryJobForTest(repo *mockRepository, disabler *fakeDisabler) *ExpiryJob {
	svc := NewService(repo, &mockTTLConfigRetriever{}, slog.Default())
	return svc.NewExpiryJob(disabler)
}

func TestExpiryJob_NoExpiredAddresses_SkipsDisabler(t *testing.T) {
	is := is.New(t)
	disabler := &fakeDisabler{}
	job := newExpiryJobForTest(newMockRepository(), disabler)

	err := job.Run(context.Background())

	is.NoErr(err)
	is.Equal(disabler.calls, 0)
}

func TestExpiryJob_ExpiredAddressesFound_DisablesAll(t *testing.T) {
	is := is.New(t)
	repo := newMockRepository()
	repo.expiredAddressIDs = []ids.AddressID{ids.AddressID(1), ids.AddressID(2)}
	disabler := &fakeDisabler{}
	job := newExpiryJobForTest(repo, disabler)

	err := job.Run(context.Background())

	is.NoErr(err)
	is.Equal(disabler.calls, 1)
	is.Equal(len(disabler.calledWith), 2)
	is.Equal(disabler.calledWith[0], ids.AddressID(1))
	is.Equal(disabler.calledWith[1], ids.AddressID(2))
	is.Equal(disabler.source, device.EventSourceExpiry)
}

func TestExpiryJob_FinderError_Propagates(t *testing.T) {
	is := is.New(t)
	finderErr := errors.New("db error")
	repo := newMockRepository()
	repo.getExpiredErr = finderErr
	disabler := &fakeDisabler{}
	job := newExpiryJobForTest(repo, disabler)

	err := job.Run(context.Background())

	is.True(errors.Is(err, finderErr))
	is.Equal(disabler.calls, 0)
}

func TestExpiryJob_DisablerError_Propagates(t *testing.T) {
	is := is.New(t)
	disablerErr := errors.New("disable error")
	repo := newMockRepository()
	repo.expiredAddressIDs = []ids.AddressID{ids.AddressID(1)}
	disabler := &fakeDisabler{err: disablerErr}
	job := newExpiryJobForTest(repo, disabler)

	err := job.Run(context.Background())

	is.True(errors.Is(err, disablerErr))
}
