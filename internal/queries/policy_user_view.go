package queries

import (
	"cmp"
	"slices"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

// policyAuditUserRow is a single non-deleted user row returned by getAllUsersForPolicyAudit.
type policyAuditUserRow struct {
	UserID          auth.UserID `db:"user_id"`
	UserName        string      `db:"user_name"`
	Username        string      `db:"username"`
	IsAdmin         bool        `db:"is_admin"`
	BypassAllowlist bool        `db:"bypass_allowlist"`
}

// ipBucket holds intermediate state for one (user, IP) cell during index assembly.
type ipBucket struct {
	// entryBypass is entry.BypassAllowlist: the IP-level bypass flag.
	entryBypass bool
	// entryAllowedHosts is the post-intersection allowed host set for this IP entry.
	entryAllowedHosts []string
	// userBypass is the per-user bypass flag — invariant per user across all devices.
	userBypass bool
	// userAllowedHosts is the user's pre-intersection host list — invariant per user across all devices.
	userAllowedHosts []string
	// addresses are the user's own addresses at this IP.
	addresses []httpapi.PolicyUserAddress
}

// buildIPIndex projects the cache snapshot + enrichment into two indexes:
//   - byUser:    userID → ip → ipBucket
//   - usersAtIP: ip → set of userIDs
//
// Addresses absent from addressEnrichment (deleted / unknown) are skipped.
func buildIPIndex(
	snap policy.PolicyMapSnapshot,
	addressEnrichment map[device.AddressID]policyEnrichmentRow,
) (byUser map[auth.UserID]map[string]*ipBucket, usersAtIP map[string]map[auth.UserID]struct{}) {
	byUser = make(map[auth.UserID]map[string]*ipBucket)
	usersAtIP = make(map[string]map[auth.UserID]struct{})

	for _, entry := range snap.Entries {
		ip := entry.IP
		if usersAtIP[ip] == nil {
			usersAtIP[ip] = make(map[auth.UserID]struct{})
		}

		for _, c := range entry.Contributors {
			meta, ok := addressEnrichment[c.AddressID]
			if !ok {
				continue
			}

			userID := c.UserID
			usersAtIP[ip][userID] = struct{}{}

			if byUser[userID] == nil {
				byUser[userID] = make(map[string]*ipBucket)
			}
			bucket := byUser[userID][ip]
			if bucket == nil {
				// userBypass and userAllowedHosts are user-level invariants: identical
				// for every ContributorAccess sharing the same UserID.
				bucket = &ipBucket{
					entryBypass:       entry.BypassAllowlist,
					entryAllowedHosts: entry.AllowedHosts,
					userBypass:        c.UserBypass,
					userAllowedHosts:  c.UserAllowedHosts,
				}
				byUser[userID][ip] = bucket
			}

			addr := httpapi.PolicyUserAddress{
				AddressId:  c.AddressID.Int64(),
				DeviceId:   c.DeviceID.Int64(),
				DeviceName: meta.DeviceName,
				UpdatedAt:  httpapi.UTCTime(meta.AddressUpdatedAt),
			}
			bucket.addresses = append(bucket.addresses, addr)
		}
	}
	return byUser, usersAtIP
}

// assemblePolicyUserMap is a pure function — no I/O, no DB, no context.
// It projects the cache snapshot + enrichment + user list into the user-pivoted
// PolicyUserMapAudit DTO. This is the unit-test target.
func assemblePolicyUserMap(
	snap policy.PolicyMapSnapshot,
	addressEnrichment map[device.AddressID]policyEnrichmentRow,
	allUsers []policyAuditUserRow, // all non-deleted users, ORDER BY display_name, id
	allowedHostsByUser map[auth.UserID][]string, // fallback host list for users absent from the cache
) httpapi.PolicyUserMapAudit {
	byUser, usersAtIP := buildIPIndex(snap, addressEnrichment)

	// Build a lookup so buildUserIPs can annotate shared-user entries with names.
	userInfoByID := make(map[auth.UserID]policyAuditUserRow, len(allUsers))
	for _, ur := range allUsers {
		userInfoByID[ur.UserID] = ur
	}

	// Aggregate: distinct hosts across the union of all users' pre-bypass host lists.
	allHostsSet := make(map[string]struct{})

	users := make([]httpapi.PolicyUserEntry, 0, len(allUsers))

	for _, ur := range allUsers {
		userID := ur.UserID
		ipMap, present := byUser[userID]

		// Resolve pre-intersection host list.
		// Cached users:    sourced from ContributorAccess (cache-consistent).
		// No-access users: sourced from the DB query (only available source).
		var userHosts []string
		if present {
			for _, b := range ipMap {
				userHosts = b.userAllowedHosts
				break
			}
		} else {
			userHosts = allowedHostsByUser[userID]
		}
		if userHosts == nil {
			userHosts = []string{}
		}

		// Collect into the global host union before the bypass override zeroes it.
		for _, h := range userHosts {
			allHostsSet[h] = struct{}{}
		}

		allowedHostCount := len(userHosts)
		if ur.BypassAllowlist {
			allowedHostCount = 0
			userHosts = []string{}
		}

		if !present {
			users = append(users, httpapi.PolicyUserEntry{
				UserId:              userID.Int64(),
				UserName:            ur.UserName,
				IsAdmin:             ur.IsAdmin,
				BypassAllowlist:     ur.BypassAllowlist,
				OnSharedIp:          false,
				IntersectionApplied: false,
				DeviceCount:         0,
				IpCount:             0,
				AllowedHostCount:    allowedHostCount,
				LastSeenAt:          nil,
				UserAllowedHosts:    userHosts,
				Ips:                 []httpapi.PolicyUserIP{},
			})
			continue
		}

		ips := buildUserIPs(userID, ipMap, usersAtIP, byUser, userInfoByID)

		users = append(users, httpapi.PolicyUserEntry{
			UserId:              userID.Int64(),
			UserName:            ur.UserName,
			IsAdmin:             ur.IsAdmin,
			BypassAllowlist:     ur.BypassAllowlist,
			OnSharedIp:          anySharedIP(ips),
			IntersectionApplied: anyIntersection(ips),
			DeviceCount:         countDistinctDevices(ipMap),
			IpCount:             len(ips),
			AllowedHostCount:    allowedHostCount,
			LastSeenAt:          maxLastSeenAt(ipMap),
			UserAllowedHosts:    userHosts,
			Ips:                 ips,
		})
	}

	// Compute top-level aggregates from usersAtIP and the assembled users slice.
	totalDeviceCount := 0
	for _, u := range users {
		totalDeviceCount += u.DeviceCount
	}

	sharedIPCount := 0
	for _, uids := range usersAtIP {
		if len(uids) >= 2 {
			sharedIPCount++
		}
	}

	return httpapi.PolicyUserMapAudit{
		RefreshedAt:       httpapi.UTCTime(snap.LastRefreshedAt),
		RefreshDurationMs: int(snap.LastRefreshDurationMs),
		TotalIpCount:      len(usersAtIP),
		TotalDeviceCount:  totalDeviceCount,
		TotalHostCount:    len(allHostsSet),
		SharedIpCount:     sharedIPCount,
		Users:             users,
	}
}

// buildUserIPs converts one user's ipMap into a sorted []PolicyUserIP slice.
// IPs are sorted lexicographically; addresses within each IP are sorted by address_id.
func buildUserIPs(
	userID auth.UserID,
	ipMap map[string]*ipBucket,
	usersAtIP map[string]map[auth.UserID]struct{},
	byUser map[auth.UserID]map[string]*ipBucket,
	userInfoByID map[auth.UserID]policyAuditUserRow,
) []httpapi.PolicyUserIP {
	sortedIPs := make([]string, 0, len(ipMap))
	for ip := range ipMap {
		sortedIPs = append(sortedIPs, ip)
	}
	slices.Sort(sortedIPs)

	result := make([]httpapi.PolicyUserIP, 0, len(sortedIPs))
	for _, ip := range sortedIPs {
		bucket := ipMap[ip]

		// Build enriched shared-user entries: one entry per co-located user (excl. self),
		// sorted by user_id for stability, each carrying their devices at this IP.
		sharedUserIDs := make([]auth.UserID, 0)
		for uid := range usersAtIP[ip] {
			if uid != userID {
				sharedUserIDs = append(sharedUserIDs, uid)
			}
		}
		slices.SortFunc(sharedUserIDs, func(a, b auth.UserID) int {
			return cmp.Compare(a, b)
		})

		sharedUsers := make([]httpapi.PolicyUserIPSharedUser, 0, len(sharedUserIDs))
		for _, uid := range sharedUserIDs {
			info := userInfoByID[uid]
			devices := make([]httpapi.PolicyIPDevice, 0)
			if otherBucket := byUser[uid][ip]; otherBucket != nil {
				seen := make(map[httpapi.ID]struct{})
				for _, addr := range otherBucket.addresses {
					if _, ok := seen[addr.DeviceId]; ok {
						continue
					}
					seen[addr.DeviceId] = struct{}{}
					devices = append(devices, httpapi.PolicyIPDevice{
						DeviceId:   addr.DeviceId,
						DeviceName: addr.DeviceName,
					})
				}
				slices.SortFunc(devices, func(a, b httpapi.PolicyIPDevice) int {
					return cmp.Compare(a.DeviceId, b.DeviceId)
				})
			}
			sharedUsers = append(sharedUsers, httpapi.PolicyUserIPSharedUser{
				UserId:   uid.Int64(),
				Username: info.Username,
				UserName: info.UserName,
				Devices:  devices,
			})
		}

		// Compute effective_hosts and trimmed_hosts.
		var effectiveHosts, trimmedHosts []string
		if bucket.userBypass || bucket.entryBypass {
			// Bypass user OR full-IP bypass: no host restrictions.
			effectiveHosts = []string{}
			trimmedHosts = []string{}
		} else {
			// effective = user's hosts ∩ entry's post-intersection hosts
			// trimmed  = user's hosts \ entry's post-intersection hosts
			effectiveHosts = sortedIntersect(bucket.userAllowedHosts, bucket.entryAllowedHosts)
			trimmedHosts = sortedDiff(bucket.userAllowedHosts, bucket.entryAllowedHosts)
		}

		// Sort addresses by address_id for stable diffing.
		addrs := make([]httpapi.PolicyUserAddress, len(bucket.addresses))
		copy(addrs, bucket.addresses)
		slices.SortFunc(addrs, func(a, b httpapi.PolicyUserAddress) int {
			return cmp.Compare(a.AddressId, b.AddressId)
		})

		result = append(result, httpapi.PolicyUserIP{
			Ip:              ip,
			SharedWithUsers: sharedUsers,
			BypassAtIp:      bucket.entryBypass,
			EffectiveHosts:  effectiveHosts,
			TrimmedHosts:    trimmedHosts,
			Addresses:       addrs,
		})
	}
	return result
}

// countDistinctDevices counts distinct device IDs across all addresses for a user.
func countDistinctDevices(ipMap map[string]*ipBucket) int {
	seen := make(map[httpapi.ID]struct{})
	for _, bucket := range ipMap {
		for _, addr := range bucket.addresses {
			seen[addr.DeviceId] = struct{}{}
		}
	}
	return len(seen)
}

// anySharedIP returns true if any IP entry has co-located users.
func anySharedIP(ips []httpapi.PolicyUserIP) bool {
	for _, ip := range ips {
		if len(ip.SharedWithUsers) > 0 {
			return true
		}
	}
	return false
}

// anyIntersection returns true if any IP entry has non-empty trimmed_hosts.
func anyIntersection(ips []httpapi.PolicyUserIP) bool {
	for _, ip := range ips {
		if len(ip.TrimmedHosts) > 0 {
			return true
		}
	}
	return false
}

// maxLastSeenAt returns the most recent address.UpdatedAt across all buckets,
// or nil if there are no addresses.
func maxLastSeenAt(ipMap map[string]*ipBucket) *httpapi.UTCTime {
	var max time.Time
	for _, bucket := range ipMap {
		for _, addr := range bucket.addresses {
			t := time.Time(addr.UpdatedAt)
			if t.After(max) {
				max = t
			}
		}
	}
	if max.IsZero() {
		return nil
	}
	v := httpapi.UTCTime(max)
	return &v
}

// sortedIntersect returns the elements present in both a and b.
// Both slices must be sorted lexicographically. The result is sorted.
func sortedIntersect(a, b []string) []string {
	result := make([]string, 0)
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			result = append(result, a[i])
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return result
}

// sortedDiff returns elements in a that are NOT present in b.
// Both slices must be sorted lexicographically. The result is sorted.
func sortedDiff(a, b []string) []string {
	result := make([]string, 0)
	i, j := 0, 0
	for i < len(a) {
		if j >= len(b) || a[i] < b[j] {
			result = append(result, a[i])
			i++
		} else if a[i] == b[j] {
			i++
			j++
		} else {
			j++
		}
	}
	return result
}
