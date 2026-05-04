//go:build test

package queries_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// stubPolicyMapReader is a minimal PolicyMapReader for integration tests.
type stubPolicyMapReader struct {
	snap policy.PolicyMapSnapshot
}

func (s *stubPolicyMapReader) GetPolicyMap() policy.PolicyMapSnapshot {
	return s.snap
}

// TestBuildPolicyUserMap_NoAccessUser verifies that non-deleted users absent
// from the snapshot appear with empty ips and populated bypass/host fields.
func TestBuildPolicyUserMap_NoAccessUser(t *testing.T) {
	is := is.New(t)

	srv := testutils.SetupIntegrationServer(t)
	repo := queries.NewRepository(srv.Database.DB())

	// The admin user exists from the seed. The snapshot is empty.
	reader := &stubPolicyMapReader{snap: policy.PolicyMapSnapshot{
		LastRefreshedAt:       time.Now().UTC(),
		LastRefreshDurationMs: 0,
	}}

	result, err := repo.BuildPolicyUserMap(t.Context(), reader)
	is.NoErr(err)

	// At least the seeded admin user must appear.
	is.True(len(result.Users) >= 1)
	adminFound := false
	for _, u := range result.Users {
		is.Equal(len(u.Ips), 0)
		is.True(u.LastSeenAt == nil)
		// Required slices must not be nil (JSON must serialize as []).
		is.True(u.Ips != nil)
		is.True(u.UserAllowedHosts != nil)
		if u.IsAdmin {
			adminFound = true
		}
	}
	// The seeded superadmin must be flagged as admin.
	is.True(adminFound)

	// Aggregates must be present (empty snapshot → all zero/empty).
	is.Equal(result.TotalIpCount, 0)
	is.Equal(result.TotalDeviceCount, 0)
	is.Equal(result.SharedIpCount, 0)
}

// TestBuildPolicyUserMap_CachePresentUser verifies that a user with a registered
// address in the snapshot has IP entries populated after BuildPolicyUserMap.
func TestBuildPolicyUserMap_CachePresentUser(t *testing.T) {
	is := is.New(t)

	srv := testutils.SetupIntegrationServer(t)
	repo := queries.NewRepository(srv.Database.DB())

	principal := testutils.AdminPrincipal(t, srv)
	adminID := principal.UserID

	// Create a device + address for the admin user.
	dev, err := srv.DeviceService.CreateDevice(t.Context(), principal, "audit-integration-device", nil)
	is.NoErr(err)
	addr, _, err := srv.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "192.168.1.50", device.EventSourceManual)
	is.NoErr(err)

	// Build a stub snapshot with one entry for that address.
	snap := policy.PolicyMapSnapshot{
		LastRefreshedAt:       time.Now().UTC(),
		LastRefreshDurationMs: 5,
		Entries: []policy.PolicyMapEntry{
			{
				IP:           "192.168.1.50",
				AllowedHosts: []string{},
				Contributors: []policy.ContributorAccess{
					{
						DeviceID:         dev.ID,
						AddressID:        addr.ID,
						UserID:           auth.UserID(adminID),
						UserBypass:       false,
						UserAllowedHosts: []string{},
					},
				},
			},
		},
	}

	result, err := repo.BuildPolicyUserMap(t.Context(), &stubPolicyMapReader{snap: snap})
	is.NoErr(err)

	// Find the admin user.
	adminIdx := -1
	for i := range result.Users {
		if result.Users[i].UserId == int64(adminID) {
			adminIdx = i
			break
		}
	}
	is.True(adminIdx >= 0)
	u := result.Users[adminIdx]
	is.Equal(u.IpCount, 1)
	is.Equal(u.DeviceCount, 1)
	is.Equal(len(u.Ips), 1)
	is.Equal(u.Ips[0].Ip, "192.168.1.50")
	is.True(u.LastSeenAt != nil)
}

// TestBuildPolicyUserMap_UsersSortedAlphabetically verifies that the users array
// is sorted by display name ascending.
func TestBuildPolicyUserMap_UsersSortedAlphabetically(t *testing.T) {
	is := is.New(t)

	srv := testutils.SetupIntegrationServer(t)
	repo := queries.NewRepository(srv.Database.DB())

	// Create a second user whose display name sorts before "admin".
	adminPrincipal := testutils.AdminPrincipal(t, srv)
	_, err := srv.AuthService.CreateUser(t.Context(), "aardvark", "Aardvark User", "aardvark@test.local", adminPrincipal)
	is.NoErr(err)

	reader := &stubPolicyMapReader{snap: policy.PolicyMapSnapshot{LastRefreshedAt: time.Now().UTC()}}

	result, err := repo.BuildPolicyUserMap(t.Context(), reader)
	is.NoErr(err)

	is.True(len(result.Users) >= 2)
	for i := 1; i < len(result.Users); i++ {
		prev := result.Users[i-1].UserName
		curr := result.Users[i].UserName
		is.True(prev <= curr)
	}
}
