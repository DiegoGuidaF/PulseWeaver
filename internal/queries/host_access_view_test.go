//go:build test

package queries_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

// ── GetHostGroupsDetails helpers ──────────────────────────────────────────────

func insertTestHostGroup(t *testing.T, db *database.DB, name string) ids.HostGroupID {
	t.Helper()
	var id ids.HostGroupID
	if err := db.QueryRowxContext(t.Context(),
		`INSERT INTO host_groups (name, color, icon) VALUES (?, '', '') RETURNING id`, name,
	).Scan(&id); err != nil {
		t.Fatalf("insertTestHostGroup(%q): %v", name, err)
	}
	return id
}

func insertTestHost(t *testing.T, db *database.DB, fqdn string) ids.HostID {
	t.Helper()
	var id ids.HostID
	if err := db.QueryRowxContext(t.Context(),
		`INSERT INTO hosts (fqdn) VALUES (?) RETURNING id`, fqdn,
	).Scan(&id); err != nil {
		t.Fatalf("insertTestHost(%q): %v", fqdn, err)
	}
	return id
}

func insertTestUserRaw(t *testing.T, db *database.DB, username string, deleted bool) ids.UserID {
	t.Helper()
	var id ids.UserID
	deletedExpr := "NULL"
	if deleted {
		deletedExpr = "'2024-01-01 00:00:00'"
	}
	q := fmt.Sprintf(
		`INSERT INTO users (username, display_name, password_hash, role, deleted_at) VALUES (?, ?, NULL, 'user', %s) RETURNING id`,
		deletedExpr,
	)
	if err := db.QueryRowxContext(t.Context(), q, username, username).Scan(&id); err != nil {
		t.Fatalf("insertTestUserRaw(%q): %v", username, err)
	}
	return id
}

func addHostToGroup(t *testing.T, db *database.DB, groupID ids.HostGroupID, hostID ids.HostID) {
	t.Helper()
	if _, err := db.ExecContext(t.Context(),
		`INSERT INTO host_group_members (host_group_id, host_id) VALUES (?, ?)`, groupID, hostID,
	); err != nil {
		t.Fatalf("addHostToGroup: %v", err)
	}
}

func grantUserToGroup(t *testing.T, db *database.DB, userID ids.UserID, groupID ids.HostGroupID) {
	t.Helper()
	if _, err := db.ExecContext(t.Context(),
		`INSERT INTO user_allowed_host_groups (user_id, host_group_id) VALUES (?, ?)`, userID, groupID,
	); err != nil {
		t.Fatalf("grantUserToGroup: %v", err)
	}
}

func setUserBypass(t *testing.T, db *database.DB, userID ids.UserID, bypass bool) {
	t.Helper()
	if _, err := db.ExecContext(t.Context(),
		`INSERT INTO user_host_settings (user_id, bypass_host_check) VALUES (?, ?)
		 ON CONFLICT (user_id) DO UPDATE SET bypass_host_check = excluded.bypass_host_check`,
		userID, bypass,
	); err != nil {
		t.Fatalf("setUserBypass: %v", err)
	}
}

// ── GetHostGroupsDetails ──────────────────────────────────────────────────────

func TestRepository_GetHostGroupsDetails_EmptyGroups(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	result, err := repos.queries.GetHostGroupsDetails(t.Context())
	is.NoErr(err)
	is.Equal(len(result.Groups), 0)
}

func TestRepository_GetHostGroupsDetails_GroupWithHosts(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	groupID := insertTestHostGroup(t, repos.db, "backend")
	h1 := insertTestHost(t, repos.db, "api.example.com")
	h2 := insertTestHost(t, repos.db, "db.example.com")
	addHostToGroup(t, repos.db, groupID, h1)
	addHostToGroup(t, repos.db, groupID, h2)

	result, err := repos.queries.GetHostGroupsDetails(t.Context())
	is.NoErr(err)
	is.Equal(len(result.Groups), 1)

	g := result.Groups[0]
	is.Equal(g.Name, "backend")
	is.Equal(len(g.Hosts), 2)
	is.True(g.Users != nil)
	is.Equal(len(*g.Users), 0)
}

func TestRepository_GetHostGroupsDetails_GroupWithUsers(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	groupID := insertTestHostGroup(t, repos.db, "devs")
	alice := insertTestUserRaw(t, repos.db, "alice", false)
	bob := insertTestUserRaw(t, repos.db, "bob", false)
	grantUserToGroup(t, repos.db, alice, groupID)
	grantUserToGroup(t, repos.db, bob, groupID)

	result, err := repos.queries.GetHostGroupsDetails(t.Context())
	is.NoErr(err)
	is.Equal(len(result.Groups), 1)

	g := result.Groups[0]
	is.Equal(len(g.Hosts), 0)
	is.True(g.Users != nil)
	is.Equal(len(*g.Users), 2)
}

func TestRepository_GetHostGroupsDetails_DeletedUserExcluded(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	groupID := insertTestHostGroup(t, repos.db, "mixed")
	active := insertTestUserRaw(t, repos.db, "active-user", false)
	deleted := insertTestUserRaw(t, repos.db, "deleted-user", true)
	grantUserToGroup(t, repos.db, active, groupID)
	grantUserToGroup(t, repos.db, deleted, groupID)

	result, err := repos.queries.GetHostGroupsDetails(t.Context())
	is.NoErr(err)
	is.Equal(len(result.Groups), 1)

	users := *result.Groups[0].Users
	is.Equal(len(users), 1)
	is.Equal(users[0].Username, "active-user")
}

func TestRepository_GetHostGroupsDetails_MultipleGroupsIsolated(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	g1 := insertTestHostGroup(t, repos.db, "group-hosts")
	g2 := insertTestHostGroup(t, repos.db, "group-users")
	host := insertTestHost(t, repos.db, "host.example.com")
	user := insertTestUserRaw(t, repos.db, "charlie", false)

	addHostToGroup(t, repos.db, g1, host)
	grantUserToGroup(t, repos.db, user, g2)

	result, err := repos.queries.GetHostGroupsDetails(t.Context())
	is.NoErr(err)
	is.Equal(len(result.Groups), 2)

	byName := make(map[string]int)
	for i, g := range result.Groups {
		byName[g.Name] = i
	}

	g1Result := result.Groups[byName["group-hosts"]]
	is.Equal(len(g1Result.Hosts), 1)
	is.Equal(g1Result.Hosts[0].Fqdn, "host.example.com")
	is.Equal(len(*g1Result.Users), 0)

	g2Result := result.Groups[byName["group-users"]]
	is.Equal(len(g2Result.Hosts), 0)
	is.Equal(len(*g2Result.Users), 1)
	is.Equal((*g2Result.Users)[0].Username, "charlie")
}

func TestRepository_GetHostGroupsDetails_UpdatedAtPopulated(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	insertTestHostGroup(t, repos.db, "timestamped")

	result, err := repos.queries.GetHostGroupsDetails(t.Context())
	is.NoErr(err)
	is.Equal(len(result.Groups), 1)
	is.True(!time.Time(result.Groups[0].UpdatedAt).IsZero())
	is.True(!time.Time(result.Groups[0].CreatedAt).IsZero())
}

// ── GetHostGroupsDetails: bypass reach (effective access beyond explicit grants) ──────────────

func TestRepository_GetHostGroupsDetails_BypassSubjectCount_UsersAndPolicies(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	groupID := insertTestHostGroup(t, repos.db, "media")

	// alice: explicit group grant, no bypass — counted in users, not in bypass
	alice := insertTestUserRaw(t, repos.db, "alice", false)
	grantUserToGroup(t, repos.db, alice, groupID)

	// charlie & diana: reach every host via bypass, regardless of group membership
	charlie := insertTestUserRaw(t, repos.db, "charlie", false)
	setUserBypass(t, repos.db, charlie, true)
	diana := insertTestUserRaw(t, repos.db, "diana", false)
	setUserBypass(t, repos.db, diana, true)

	// corp-vpn: bypass policy, reaches every host regardless of group assignment
	insertTestPolicy(t, repos.db, "corp-vpn", "10.0.0.0/8", true)
	// scoped-net: non-bypass policy, explicitly assigned to this group
	scoped := insertTestPolicy(t, repos.db, "scoped-net", "192.168.1.0/24", false)
	assignGroupToPolicy(t, repos.db, scoped, groupID)

	result, err := repos.queries.GetHostGroupsDetails(t.Context())
	is.NoErr(err)
	is.Equal(len(result.Groups), 1)

	g := result.Groups[0]
	is.Equal(len(*g.Users), 1)          // alice only (explicit grant)
	is.Equal(len(g.NetworkPolicies), 1) // scoped-net only (explicit assignment)
	is.Equal(g.BypassSubjectCount, 3)   // charlie + diana + corp-vpn
}

func TestRepository_GetHostGroupsDetails_BypassSubjectCount_ExcludesAlreadyGranted(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	groupID := insertTestHostGroup(t, repos.db, "shared")

	// charlie has both an explicit group grant AND bypass — must not be double counted.
	charlie := insertTestUserRaw(t, repos.db, "charlie", false)
	grantUserToGroup(t, repos.db, charlie, groupID)
	setUserBypass(t, repos.db, charlie, true)

	// corp-vpn is both assigned to the group AND a bypass policy — must not be double counted.
	corpVPN := insertTestPolicy(t, repos.db, "corp-vpn", "10.0.0.0/8", true)
	assignGroupToPolicy(t, repos.db, corpVPN, groupID)

	result, err := repos.queries.GetHostGroupsDetails(t.Context())
	is.NoErr(err)
	is.Equal(len(result.Groups), 1)

	g := result.Groups[0]
	is.Equal(len(*g.Users), 1)
	is.Equal(len(g.NetworkPolicies), 1)
	is.Equal(g.BypassSubjectCount, 0) // both bypass subjects already listed as explicit grants
}

func TestRepository_GetHostGroupsDetails_BypassSubjectCount_ZeroWhenNoBypass(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	groupID := insertTestHostGroup(t, repos.db, "plain")
	alice := insertTestUserRaw(t, repos.db, "alice", false)
	grantUserToGroup(t, repos.db, alice, groupID)
	setUserBypass(t, repos.db, alice, false)
	insertTestPolicy(t, repos.db, "scoped-net", "192.168.1.0/24", false)

	result, err := repos.queries.GetHostGroupsDetails(t.Context())
	is.NoErr(err)
	is.Equal(len(result.Groups), 1)
	is.Equal(result.Groups[0].BypassSubjectCount, 0)
}
