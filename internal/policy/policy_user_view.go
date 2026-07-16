package policy

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/DiegoGuidaF/PulseWeaver/internal/slicex"
	"github.com/jmoiron/sqlx"
)

// PolicyMapReader is the consumer-side interface the policy-map view reads
// the cache snapshot through. *Service satisfies it via GetPolicyMap.
type PolicyMapReader interface {
	GetPolicyMap() PolicyMapSnapshot
}

// policyAuditUserRow is a single non-deleted user row returned by getAllUsersForPolicyAudit.
type policyAuditUserRow struct {
	UserID          ids.UserID `db:"user_id"`
	UserName        string     `db:"user_name"`
	Username        string     `db:"username"`
	IsAdmin         bool       `db:"is_admin"`
	BypassAllowlist bool       `db:"bypass_allowlist"`
}

// policyEnrichmentRow holds the address metadata fetched by getPolicyAddressEnrichment.
// DeviceID and UserID are sourced from the policy snapshot's ContributorAccess and are
// not re-fetched here.
type policyEnrichmentRow struct {
	AddressID        ids.AddressID `db:"address_id"`
	AddressUpdatedAt time.Time     `db:"address_updated_at"`
	DeviceName       string        `db:"device_name"`
}

// userIPIndex maps userID → ip → ipBucket.
type userIPIndex map[ids.UserID]map[string]*ipBucket

// ipUsersIndex maps ip → set of userIDs present at that IP.
type ipUsersIndex map[string]map[ids.UserID]struct{}

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
	snap PolicyMapSnapshot,
	addressEnrichment map[ids.AddressID]policyEnrichmentRow,
) (byUser userIPIndex, usersAtIP ipUsersIndex) {
	byUser = make(userIPIndex)
	usersAtIP = make(ipUsersIndex)

	for _, entry := range snap.Entries {
		ip := entry.IP
		if usersAtIP[ip] == nil {
			usersAtIP[ip] = make(map[ids.UserID]struct{})
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
// PolicyUserMapAudit DTO.
func assemblePolicyUserMap(
	snap PolicyMapSnapshot,
	addressEnrichment map[ids.AddressID]policyEnrichmentRow,
	allUsers []policyAuditUserRow, // all non-deleted users, ORDER BY display_name, id
	allowedHostsByUser map[ids.UserID][]string, // fallback host list for users absent from the cache
) httpapi.PolicyUserMapAudit {
	byUser, usersAtIP := buildIPIndex(snap, addressEnrichment)

	// Build a lookup so buildUserIPs can annotate shared-user entries with names.
	userInfoByID := make(map[ids.UserID]policyAuditUserRow, len(allUsers))
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
				UserId:           userID.Int64(),
				DisplayName:      ur.UserName,
				IsAdmin:          ur.IsAdmin,
				BypassAllowlist:  ur.BypassAllowlist,
				Status:           DeriveUserStatus(ur.BypassAllowlist, 0, allowedHostCount),
				OnSharedIp:       false,
				DeviceCount:      0,
				IpCount:          0,
				AllowedHostCount: allowedHostCount,
				LastSeenAt:       nil,
				UserAllowedHosts: userHosts,
				Ips:              []httpapi.PolicyUserIP{},
			})
			continue
		}

		ips := buildUserIPs(userID, ipMap, usersAtIP, byUser, userInfoByID)

		users = append(users, httpapi.PolicyUserEntry{
			UserId:           userID.Int64(),
			DisplayName:      ur.UserName,
			IsAdmin:          ur.IsAdmin,
			BypassAllowlist:  ur.BypassAllowlist,
			Status:           DeriveUserStatus(ur.BypassAllowlist, len(ips), allowedHostCount),
			OnSharedIp:       anySharedIP(ips),
			DeviceCount:      countDistinctDevices(ipMap),
			IpCount:          len(ips),
			AllowedHostCount: allowedHostCount,
			LastSeenAt:       maxLastSeenAt(ipMap),
			UserAllowedHosts: userHosts,
			Ips:              ips,
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
	userID ids.UserID,
	ipMap map[string]*ipBucket,
	usersAtIP ipUsersIndex,
	byUser userIPIndex,
	userInfoByID map[ids.UserID]policyAuditUserRow,
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
		sharedUserIDs := make([]ids.UserID, 0)
		for uid := range usersAtIP[ip] {
			if uid != userID {
				sharedUserIDs = append(sharedUserIDs, uid)
			}
		}
		slices.SortFunc(sharedUserIDs, func(a, b ids.UserID) int {
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
			effectiveHosts = slicex.Intersect(bucket.userAllowedHosts, bucket.entryAllowedHosts)
			trimmedHosts = slicex.Diff(bucket.userAllowedHosts, bucket.entryAllowedHosts)
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

// enrichGeo resolves each IP's GeoInfo in place. Pure post-processing over the
// assembled view; a nil resolver leaves every entry's geo unset.
func enrichGeo(ips []httpapi.PolicyUserIP, geo GeoIPResolver) {
	if geo == nil {
		return
	}
	for i := range ips {
		ips[i].Geo = geoInfoFromResult(geo.Resolve(ips[i].Ip))
	}
}

// geoInfoFromResult maps a geoip.Result to the API GeoInfo DTO, returning nil
// when the lookup found nothing so the field is omitted from the response.
func geoInfoFromResult(r geoip.Result) *httpapi.GeoInfo {
	if r.IsEmpty() {
		return nil
	}
	info := &httpapi.GeoInfo{}
	if r.CountryCode != "" {
		info.CountryCode = &r.CountryCode
	}
	if r.CountryName != "" {
		info.CountryName = &r.CountryName
	}
	if r.ContinentCode != "" {
		info.ContinentCode = &r.ContinentCode
	}
	if r.ASN != 0 {
		asn := int64(r.ASN)
		info.Asn = &asn
	}
	if r.ASNOrg != "" {
		info.AsnOrg = &r.ASNOrg
	}
	return info
}

// DeriveUserStatus classifies a user along two orthogonal axes — reachability
// (live IPs in the cache) and authorization (host grants) — into the status the
// audit view renders. Bypass short-circuits: the host allowlist does not apply.
func DeriveUserStatus(bypass bool, ipCount, allowedHostCount int) httpapi.PolicyUserStatus {
	if bypass {
		return httpapi.Bypass
	}
	hasLiveIPs := ipCount > 0
	hasHostGrants := allowedHostCount > 0
	switch {
	case hasLiveIPs && hasHostGrants:
		return httpapi.LiveWithAccess
	case hasLiveIPs && !hasHostGrants:
		return httpapi.LiveNoHostAccess
	case !hasLiveIPs && hasHostGrants:
		return httpapi.NoLiveIps
	default:
		return httpapi.NoAccess
	}
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

// getPolicyAddressEnrichment fetches display metadata for the given address IDs.
func (r *Repository) getPolicyAddressEnrichment(ctx context.Context, addressIDs []ids.AddressID) (map[ids.AddressID]policyEnrichmentRow, error) {
	if len(addressIDs) == 0 {
		return map[ids.AddressID]policyEnrichmentRow{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT
			a.id         AS address_id,
			a.updated_at AS address_updated_at,
			d.name       AS device_name
		FROM addresses a
		JOIN devices d ON d.id = a.device_id
		WHERE a.id IN (?)`, addressIDs)
	if err != nil {
		return nil, fmt.Errorf("build policy audit enrichment query: %w", err)
	}
	query = r.db.Rebind(query)

	var rows []policyEnrichmentRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("get policy address enrichment: %w", err)
	}

	result := make(map[ids.AddressID]policyEnrichmentRow, len(rows))
	for _, row := range rows {
		result[row.AddressID] = row
	}
	return result, nil
}

// getAllUsersForPolicyAudit returns every non-deleted user with their bypass flag,
// plus a map of pre-intersection allowed FQDNs keyed by user ID.
// Two queries assembled in Go per queries-read-models.md pattern.
func (r *Repository) getAllUsersForPolicyAudit(ctx context.Context) ([]policyAuditUserRow, map[ids.UserID][]string, error) {
	const usersQuery = `
		SELECT u.id           AS user_id,
		       u.username     AS username,
		       u.display_name AS user_name,
		       u.role IN ('admin', 'superadmin') AS is_admin,
		       COALESCE(uhs.bypass_host_check, 0) AS bypass_allowlist
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.deleted_at IS NULL
		ORDER BY u.display_name, u.id
	`
	var userRows []policyAuditUserRow
	if err := r.db.SelectContext(ctx, &userRows, usersQuery); err != nil {
		return nil, nil, fmt.Errorf("list users for policy audit: %w", err)
	}

	const hostsQuery = `
		SELECT uahg.user_id, h.fqdn
		FROM user_allowed_host_groups uahg
		JOIN host_group_members hgm ON hgm.host_group_id = uahg.host_group_id
		JOIN hosts h ON h.id = hgm.host_id
		ORDER BY 1, 2
	`

	type hostRow struct {
		UserID ids.UserID `db:"user_id"`
		FQDN   string     `db:"fqdn"`
	}
	var hostRows []hostRow
	if err := r.db.SelectContext(ctx, &hostRows, hostsQuery); err != nil {
		return nil, nil, fmt.Errorf("list user allowed hosts for policy audit: %w", err)
	}

	allowedHostsByUser := make(map[ids.UserID][]string, len(userRows))
	for _, h := range hostRows {
		allowedHostsByUser[h.UserID] = append(allowedHostsByUser[h.UserID], h.FQDN)
	}

	return userRows, allowedHostsByUser, nil
}

// collectAddressIDs gathers all unique address IDs referenced in a snapshot.
func collectAddressIDs(snap PolicyMapSnapshot) []ids.AddressID {
	seen := make(map[ids.AddressID]struct{})
	var addressIDs []ids.AddressID
	for _, e := range snap.Entries {
		for _, c := range e.Contributors {
			if _, ok := seen[c.AddressID]; !ok {
				seen[c.AddressID] = struct{}{}
				addressIDs = append(addressIDs, c.AddressID)
			}
		}
	}
	return addressIDs
}

// BuildPolicyUserMap is the single business-logic entry point for the user-pivoted
// policy audit view. The handler is a thin wrapper around it. This is the
// integration-test target for orchestration; pure assembly is tested separately
// via assemblePolicyUserMap.
func (r *Repository) BuildPolicyUserMap(
	ctx context.Context,
	reader PolicyMapReader,
	npProvider NetworkPoliciesProvider,
	geo GeoIPResolver,
) (httpapi.PolicyUserMapAudit, error) {
	snap := reader.GetPolicyMap()
	addressIDs := collectAddressIDs(snap)

	addrEnrichment, err := r.getPolicyAddressEnrichment(ctx, addressIDs)
	if err != nil {
		return httpapi.PolicyUserMapAudit{}, fmt.Errorf("policy address enrichment: %w", err)
	}

	allUsers, allowedHostsByUser, err := r.getAllUsersForPolicyAudit(ctx)
	if err != nil {
		return httpapi.PolicyUserMapAudit{}, fmt.Errorf("policy audit users: %w", err)
	}

	// Load enabled network policy entries for the audit view.
	var npEntries []networkpolicies.CacheEntry
	if npProvider != nil {
		npEntries, err = npProvider.GetEnabledCacheEntries(ctx)
		if err != nil {
			return httpapi.PolicyUserMapAudit{}, fmt.Errorf("policy audit network policies: %w", err)
		}
	}

	audit := assemblePolicyUserMap(snap, addrEnrichment, allUsers, allowedHostsByUser)

	for i := range audit.Users {
		enrichGeo(audit.Users[i].Ips, geo)
	}

	// Attach network policy data. The denominator is the true number of hosts in
	// the system, not audit.TotalHostCount (which is the union of users' allowed
	// hosts and may be smaller than a policy's own reachable host set).
	npAPIEntries := make([]httpapi.PolicyNetworkPolicyEntry, 0, len(npEntries))
	var totalHostCount int
	if err := r.db.GetContext(ctx, &totalHostCount, `SELECT COUNT(*) FROM hosts`); err != nil {
		return httpapi.PolicyUserMapAudit{}, fmt.Errorf("policy audit total hosts: %w", err)
	}
	for _, e := range npEntries {
		effective := len(e.AllowedHostFQDNs)
		if e.BypassHostCheck {
			effective = totalHostCount
		}
		npAPIEntries = append(npAPIEntries, httpapi.PolicyNetworkPolicyEntry{
			PolicyId:           e.PolicyID.Int64(),
			PolicyName:         e.PolicyName,
			Cidr:               e.CIDR,
			Enabled:            true,
			BypassHostCheck:    e.BypassHostCheck,
			EffectiveHostCount: effective,
		})
	}
	audit.NetworkPolicies = npAPIEntries
	audit.TotalNetworkPolicyCount = len(npAPIEntries)

	return audit, nil
}
