//go:build test

package hostaccess

import (
	"context"
	"errors"
	"testing"

	"github.com/matryer/is"
)

func TestService_ReconcileHostGroups_NotifiesObserversOnce(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	repo := &fakeRepo{
		hostGroups:     []HostGroup{{ID: 1, Name: "old"}},
		knownHostsByID: nil,
	}
	svc, _ := newTestService(repo)
	svc.AddUserHostAccessObserver(obs)

	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{{Name: "new"}},
	})

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_ReconcileHostGroups_DeleteThenUpdateThenCreate(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		hostGroups: []HostGroup{
			{ID: 1, Name: "to-update"},
			{ID: 2, Name: "to-delete"},
		},
	}
	svc, _ := newTestService(repo)

	id1 := HostGroupID(1)
	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{
			{ID: &id1, Name: "renamed"},
			{Name: "new-one"},
		},
	})
	is.NoErr(err)
	is.Equal(repo.callOrder, []string{"delete", "update", "create"})
}

func TestService_ReconcileHostGroups_NoOp_NoCalls(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	current := HostGroup{ID: 1, Name: "stable"}
	repo := &fakeRepo{hostGroups: []HostGroup{current}}
	svc, _ := newTestService(repo)
	svc.AddUserHostAccessObserver(obs)

	id := current.ID
	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{{ID: &id, Name: "stable"}},
	})

	is.NoErr(err)
	is.Equal(len(repo.callOrder), 0)
	is.Equal(obs.calls, 1) // observers still notified after a successful tx, even if it was a no-op
}

func TestService_ReconcileHostGroups_EmptyName_Rejected(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{}
	svc, _ := newTestService(repo)

	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{{Name: "  "}},
	})
	is.True(errors.Is(err, ErrGroupNameRequired))
	is.Equal(len(repo.callOrder), 0)
}

func TestService_ReconcileHostGroups_DuplicateID_Rejected(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{}
	svc, _ := newTestService(repo)

	id := HostGroupID(7)
	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{
			{ID: &id, Name: "a"},
			{ID: &id, Name: "b"},
		},
	})
	is.True(errors.Is(err, ErrDuplicateGroupID))
}

func TestBuildGroupReconcilePlan_CreateOnly(t *testing.T) {
	is := is.New(t)
	plan, err := buildGroupReconcilePlan(nil, []DesiredHostGroup{{Name: "new"}})
	is.NoErr(err)
	is.Equal(len(plan.toCreate), 1)
	is.Equal(plan.toCreate[0].Name, "new")
	is.Equal(len(plan.toUpdate), 0)
	is.Equal(len(plan.toDelete), 0)
}

func TestBuildGroupReconcilePlan_DeleteOnly(t *testing.T) {
	is := is.New(t)
	current := []HostGroup{{ID: 1, Name: "doomed"}}
	plan, err := buildGroupReconcilePlan(current, nil)
	is.NoErr(err)
	is.Equal(plan.toDelete, []HostGroupID{1})
}

func TestBuildGroupReconcilePlan_UpdateChanged(t *testing.T) {
	is := is.New(t)
	current := []HostGroup{{ID: 1, Name: "before"}}
	id := HostGroupID(1)
	plan, err := buildGroupReconcilePlan(current, []DesiredHostGroup{{ID: &id, Name: "after"}})
	is.NoErr(err)
	is.Equal(len(plan.toUpdate), 1)
	is.Equal(plan.toUpdate[0].Name, "after")
}

func TestBuildGroupReconcilePlan_UpdateUnchanged_SkipsUpdate(t *testing.T) {
	is := is.New(t)
	current := []HostGroup{{ID: 1, Name: "stable"}}
	id := HostGroupID(1)
	plan, err := buildGroupReconcilePlan(current, []DesiredHostGroup{{ID: &id, Name: "stable"}})
	is.NoErr(err)
	is.Equal(len(plan.toUpdate), 0)
	is.Equal(len(plan.toCreate), 0)
	is.Equal(len(plan.toDelete), 0)
}

func TestBuildGroupReconcilePlan_UnknownDesiredID_Errors(t *testing.T) {
	is := is.New(t)
	id := HostGroupID(42)
	_, err := buildGroupReconcilePlan(nil, []DesiredHostGroup{{ID: &id, Name: "ghost"}})
	is.True(errors.Is(err, ErrHostGroupNotFound))
}

func TestBuildGroupReconcilePlan_Mixed(t *testing.T) {
	is := is.New(t)
	current := []HostGroup{
		{ID: 1, Name: "keep"},
		{ID: 2, Name: "rename-me"},
		{ID: 3, Name: "remove-me"},
	}
	id1 := HostGroupID(1)
	id2 := HostGroupID(2)
	plan, err := buildGroupReconcilePlan(current, []DesiredHostGroup{
		{ID: &id1, Name: "keep"},
		{ID: &id2, Name: "renamed"},
		{Name: "fresh"},
	})
	is.NoErr(err)
	is.Equal(len(plan.toCreate), 1)
	is.Equal(plan.toCreate[0].Name, "fresh")
	is.Equal(len(plan.toUpdate), 1)
	is.Equal(plan.toUpdate[0].ID, HostGroupID(2))
	is.Equal(plan.toDelete, []HostGroupID{3})
}
