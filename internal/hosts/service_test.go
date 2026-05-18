//go:build test

package hosts

import (
	"context"
	"log/slog"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	err error

	hosts      []Host
	hostsByID  []Host
	hostGroups []HostGroup

	createHostCalls []HostDraft
	deleteHostCalls []ids.HostID
	setGroupCalls   []hostGroupSet

	createCalls []HostGroupDraft
	updateCalls []HostGroup
	deleteCalls []ids.HostGroupID
	callOrder   []string
}

var _ repository = (*fakeRepo)(nil)

func (f *fakeRepo) ListHosts(_ context.Context) ([]Host, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hosts, nil
}
func (f *fakeRepo) CreateHost(_ context.Context, draft HostDraft) (ids.HostID, error) {
	if f.err != nil {
		return ids.HostID(1), f.err
	}
	f.createHostCalls = append(f.createHostCalls, draft)
	f.callOrder = append(f.callOrder, "createHost")
	return ids.HostID(len(f.createHostCalls) + 100), nil
}
func (f *fakeRepo) DeleteHost(_ context.Context, id ids.HostID) error {
	if f.err != nil {
		return f.err
	}
	f.deleteHostCalls = append(f.deleteHostCalls, id)
	f.callOrder = append(f.callOrder, "deleteHost")
	return nil
}
func (f *fakeRepo) SetHostGroupMembership(_ context.Context, hostID ids.HostID, groupIDs []ids.HostGroupID) error {
	if f.err != nil {
		return f.err
	}
	f.setGroupCalls = append(f.setGroupCalls, hostGroupSet{HostID: hostID, GroupIDs: groupIDs})
	f.callOrder = append(f.callOrder, "setHostGroups")
	return nil
}
func (f *fakeRepo) ListHostsByIDs(_ context.Context, _ []ids.HostID) ([]Host, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hostsByID, nil
}
func (f *fakeRepo) ListHostGroups(_ context.Context) ([]HostGroup, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hostGroups, nil
}
func (f *fakeRepo) CreateHostGroup(_ context.Context, draft HostGroupDraft) (ids.HostGroupID, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.createCalls = append(f.createCalls, draft)
	f.callOrder = append(f.callOrder, "create")
	return ids.HostGroupID(len(f.createCalls) + 100), nil
}
func (f *fakeRepo) UpdateHostGroup(_ context.Context, group HostGroup) error {
	if f.err != nil {
		return f.err
	}
	f.updateCalls = append(f.updateCalls, group)
	f.callOrder = append(f.callOrder, "update")
	return nil
}
func (f *fakeRepo) DeleteHostGroup(_ context.Context, id ids.HostGroupID) error {
	if f.err != nil {
		return f.err
	}
	f.deleteCalls = append(f.deleteCalls, id)
	f.callOrder = append(f.callOrder, "delete")
	return nil
}
func (f *fakeRepo) AddIgnoredSuggestion(_ context.Context, _ string) (IgnoredHostSuggestion, error) {
	if f.err != nil {
		return IgnoredHostSuggestion{}, f.err
	}
	return IgnoredHostSuggestion{ID: 1}, nil
}
func (f *fakeRepo) RemoveIgnoredSuggestionByFQDN(_ context.Context, _ string) error { return f.err }

// fakeTransactor executes fn directly.
type fakeTransactor struct {
	calls int
}

func (f *fakeTransactor) WithinTx(_ context.Context, fn func(context.Context) error) error {
	f.calls++
	return fn(context.Background())
}

// mockObserver records calls to OnHostAccessChanged.
type mockObserver struct {
	calls int
}

var _ Observer = (*mockObserver)(nil)

func (m *mockObserver) OnHostAccessChanged(_ context.Context) { m.calls++ }

func newTestService(repo repository) *Service {
	return NewService(repo, &fakeTransactor{}, slog.New(slog.DiscardHandler))
}

// TestService_AddObserver_NilObserver_Ignored ensures nil observers are silently dropped.
func TestService_AddObserver_NilObserver_Ignored(t *testing.T) {
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(nil)
	if len(svc.observers) != 0 {
		t.Fatal("expected no observers")
	}
}
