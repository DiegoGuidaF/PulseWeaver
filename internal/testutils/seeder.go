//go:build test

package testutils

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

// ── fixture types ─────────────────────────────────────────────────────────────

// PolicyFixture describes the seeder inputs for a network policy.
// Desc is optional; zero value means no description.
type PolicyFixture struct {
	Name string
	CIDR string
	Desc string
}

// GroupFixture describes the seeder inputs for a host group.
// Color defaults to "#000000" and Icon defaults to "server" when left empty.
type GroupFixture struct {
	Name  string
	Color string
	Icon  string
}

// HostFixture describes the seeder inputs for a host.
// Groups lists the group names the host belongs to; each must match a seeded GroupFixture.
type HostFixture struct {
	FQDN   string
	Groups []string
}

// UserFixture describes the seeder inputs for a regular (non-admin) user.
// Name is used as username, display name, and email prefix (<name>@test.local).
type UserFixture struct {
	Name string
}

// DeviceLeaseRuleFixture describes an address-lease rule to enable on a device.
// Device must match the Name of a DeviceFixture seeded in the same Build call.
type DeviceLeaseRuleFixture struct {
	Device     string
	TTLSeconds int
}

// DeviceMaxActiveRuleFixture describes a max-active-addresses rule to enable on a device.
// Device must match the Name of a DeviceFixture seeded in the same Build call.
type DeviceMaxActiveRuleFixture struct {
	Device       string
	MaxAddresses int
}

// DeviceFixture describes the seeder inputs for a device.
// OwnerUser must match the Name of a UserFixture seeded in the same Build call.
type DeviceFixture struct {
	Name      string
	OwnerUser string
}

// AddressFixture describes the seeder inputs for a device address.
// Device must match the Name of a DeviceFixture seeded in the same Build call.
// ExpiresAt creates a lease for the address if set. Disabled disables the address after registration.
type AddressFixture struct {
	Device    string
	IP        string
	ExpiresAt *time.Time // if set, a lease is created with this expiry
	Disabled  bool       // if true, the address is disabled after registration
}

// AccessLogEntryFixture describes a policy decision event to be inserted into the
// access log.
//
// Use Devices to list device names whose (device_id, address_id, user_id) triples
// become access_log_contributors rows.  Each listed device must have a seeded
// AddressFixture whose IP equals ClientIP so the seeder can resolve the address ID.
// Devices and PolicyName are mutually exclusive.
//
// Use PolicyName to produce an access_log_network_policy_contributors row instead.
// PolicyName must match a preceding WithPolicy call.
//
// Leave both empty for no-contributor entries (e.g. unknown IPs denied at the gate).
type AccessLogEntryFixture struct {
	ClientIP   string
	Outcome    bool
	DenyReason *policy.DenyReason
	Devices    []string // optional device names for access_log_contributors
	PolicyName string   // optional; mutually exclusive with Devices
	TargetHost *string  // optional
}

// ── world fixture variables ───────────────────────────────────────────────────

// World fixture variables are named by their role in the test world, not by
// domain semantics. They are used by SeedFullWorld and by cross-domain query
// tests that need to assert against the seeded values without hardcoding strings.

var (
	FixturePolicyWithGroups = PolicyFixture{Name: "corp-vpn", CIDR: "10.0.0.0/8", Desc: "Corporate VPN access"}
	FixturePolicyNoGroups   = PolicyFixture{Name: "isolated", CIDR: "172.16.0.0/12"}

	FixtureGroupBackend  = GroupFixture{Name: "backend"}
	FixtureGroupFrontend = GroupFixture{Name: "frontend"}
	FixtureGroupEmpty    = GroupFixture{Name: "empty-group"}

	FixtureHostBackend1  = HostFixture{FQDN: "api1.internal", Groups: []string{FixtureGroupBackend.Name}}
	FixtureHostBackend2  = HostFixture{FQDN: "api2.internal", Groups: []string{FixtureGroupBackend.Name}}
	FixtureHostFrontend1 = HostFixture{FQDN: "web1.internal", Groups: []string{FixtureGroupFrontend.Name}}
	FixtureHostFrontend2 = HostFixture{FQDN: "web2.internal", Groups: []string{FixtureGroupFrontend.Name}}

	FixtureUserWithAccess   = UserFixture{Name: "alice"}   // backend + frontend, no bypass
	FixtureUserNoAccess     = UserFixture{Name: "bob"}     // no group access
	FixtureUserBypassAccess = UserFixture{Name: "charlie"} // backend with bypass=true

	FixtureDeviceWithOwnerAccess    = DeviceFixture{Name: "alice-laptop", OwnerUser: FixtureUserWithAccess.Name}
	FixtureDeviceWithoutOwnerAccess = DeviceFixture{Name: "bob-phone", OwnerUser: FixtureUserNoAccess.Name}
	FixtureDeviceBypassAccess       = DeviceFixture{Name: "charlie-desktop", OwnerUser: FixtureUserBypassAccess.Name}

	// Rules seeded on alice-laptop by SeedFullWorld.
	FixtureLeaseRuleAliceLaptop     = DeviceLeaseRuleFixture{Device: FixtureDeviceWithOwnerAccess.Name, TTLSeconds: 3600}
	FixtureMaxActiveRuleAliceLaptop = DeviceMaxActiveRuleFixture{Device: FixtureDeviceWithOwnerAccess.Name, MaxAddresses: 2}

	FixtureAddressAlice  = AddressFixture{Device: FixtureDeviceWithOwnerAccess.Name, IP: "10.1.0.1"}
	FixtureAddressBob    = AddressFixture{Device: FixtureDeviceWithoutOwnerAccess.Name, IP: "10.2.0.1"}
	FixtureAddressShared = AddressFixture{Device: FixtureDeviceBypassAccess.Name, IP: "10.1.0.1"} // charlie shares alice's IP

	// Five canonical access-log paths exercised by SeedFullWorld:
	// 1. allow — single contributor (device+user link)
	// 2. deny  — single contributor (host not allowed)
	// 3. deny  — no contributor (IP not registered)
	// 4. allow — multiple contributors for one entry (shared IP, two users)
	// 5. allow — network policy CIDR match (no device contributors)
	FixtureAccessLogAliceAllow         = AccessLogEntryFixture{ClientIP: "10.1.0.1", Outcome: true, Devices: []string{FixtureDeviceWithOwnerAccess.Name}}
	FixtureAccessLogBobHostDeny        = AccessLogEntryFixture{ClientIP: "10.2.0.1", Outcome: false, DenyReason: new(policy.DenyReasonHostNotAllowed), Devices: []string{FixtureDeviceWithoutOwnerAccess.Name}}
	FixtureAccessLogUnknownDeny        = AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered)}
	FixtureAccessLogSharedIPAllow      = AccessLogEntryFixture{ClientIP: "10.1.0.1", Outcome: true, Devices: []string{FixtureDeviceWithOwnerAccess.Name, FixtureDeviceBypassAccess.Name}}
	FixtureAccessLogNetworkPolicyAllow = AccessLogEntryFixture{ClientIP: "10.3.0.1", Outcome: true, PolicyName: FixturePolicyWithGroups.Name}
)

// ── relational spec types (internal) ─────────────────────────────────────────

type policyAccessSpec struct {
	policy string
	groups []string
}

type userAccessSpec struct {
	user   string
	groups []string
	bypass bool
}

// ── result ────────────────────────────────────────────────────────────────────

// SeedResult holds the IDs of every entity created by a Seeder.Build call.
// Lookup methods call t.Fatalf when the name was not seeded, preventing silent zero-ID bugs.
type SeedResult struct {
	t         *testing.T
	policies  map[string]ids.NetworkPolicyID
	groups    map[string]ids.HostGroupID
	hosts     map[string]ids.HostID
	users     map[string]ids.UserID
	devices   map[string]ids.DeviceID
	addresses map[string]ids.AddressID // keyed by addressKey(device, ip)
}

func (r *SeedResult) Policy(name string) ids.NetworkPolicyID {
	r.t.Helper()
	id, ok := r.policies[name]
	if !ok {
		r.t.Fatalf("SeedResult.Policy: no policy named %q was seeded", name)
	}
	return id
}

func (r *SeedResult) Group(name string) ids.HostGroupID {
	r.t.Helper()
	id, ok := r.groups[name]
	if !ok {
		r.t.Fatalf("SeedResult.Group: no group named %q was seeded", name)
	}
	return id
}

func (r *SeedResult) Host(fqdn string) ids.HostID {
	r.t.Helper()
	id, ok := r.hosts[fqdn]
	if !ok {
		r.t.Fatalf("SeedResult.Host: no host with FQDN %q was seeded", fqdn)
	}
	return id
}

func (r *SeedResult) User(name string) ids.UserID {
	r.t.Helper()
	id, ok := r.users[name]
	if !ok {
		r.t.Fatalf("SeedResult.User: no user named %q was seeded", name)
	}
	return id
}

func (r *SeedResult) Device(name string) ids.DeviceID {
	r.t.Helper()
	id, ok := r.devices[name]
	if !ok {
		r.t.Fatalf("SeedResult.Device: no device named %q was seeded", name)
	}
	return id
}

// Address returns the ID of the address registered for device with the given IP.
func (r *SeedResult) Address(device, ip string) ids.AddressID {
	r.t.Helper()
	id, ok := r.addresses[addressKey(device, ip)]
	if !ok {
		r.t.Fatalf("SeedResult.Address: no address %q for device %q was seeded", ip, device)
	}
	return id
}

func addressKey(device, ip string) string { return device + ":" + ip }

// ── builder ───────────────────────────────────────────────────────────────────

// Seeder declaratively describes test fixtures and materialises them in a single
// Build call. Declare entities with the fluent methods (order does not matter for
// independent entities), then call Build once to apply everything against the app.
//
// All groups are created in a single ReconcileHostGroups call and all hosts in a
// single ReconcileHosts call, so the full-replace semantics of those operations
// are handled correctly regardless of how many entities are declared.
type Seeder struct {
	t   *testing.T
	srv *app.App

	policies         []PolicyFixture
	groups           []GroupFixture
	hosts            []HostFixture
	assignments      []policyAccessSpec
	users            []UserFixture
	userAccesses     []userAccessSpec
	devices          []DeviceFixture
	leaseRules       []DeviceLeaseRuleFixture
	maxActiveRules   []DeviceMaxActiveRuleFixture
	addresses        []AddressFixture
	accessLogEntries []AccessLogEntryFixture
	initPolicy       bool
}

// NewSeeder returns a fresh Seeder bound to t and srv.
func NewSeeder(t *testing.T, srv *app.App) *Seeder {
	t.Helper()
	return &Seeder{t: t, srv: srv}
}

// WithPolicy declares a network policy. Name and CIDR are required.
// Desc is optional; leave it empty for no description.
func (s *Seeder) WithPolicy(f PolicyFixture) *Seeder {
	s.t.Helper()
	if f.Name == "" {
		s.t.Fatalf("Seeder.WithPolicy: Name is required")
	}
	if f.CIDR == "" {
		s.t.Fatalf("Seeder.WithPolicy: CIDR is required (name=%q)", f.Name)
	}
	s.policies = append(s.policies, f)
	return s
}

// WithGroup declares a host group. Name is required.
// Color defaults to "#000000" and Icon to "server" when left empty.
func (s *Seeder) WithGroup(f GroupFixture) *Seeder {
	s.t.Helper()
	if f.Name == "" {
		s.t.Fatalf("Seeder.WithGroup: Name is required")
	}
	s.groups = append(s.groups, f)
	return s
}

// WithHost declares a host and assigns it to the named groups.
// FQDN is required. Group names must match a preceding WithGroup call.
func (s *Seeder) WithHost(f HostFixture) *Seeder {
	s.t.Helper()
	if f.FQDN == "" {
		s.t.Fatalf("Seeder.WithHost: FQDN is required")
	}
	s.hosts = append(s.hosts, f)
	return s
}

// AssignGroupsToPolicy records that the named groups should be granted access to
// the named policy. Group and policy names must match preceding declarations.
func (s *Seeder) AssignGroupsToPolicy(policy string, groups ...string) *Seeder {
	s.assignments = append(s.assignments, policyAccessSpec{policy: policy, groups: groups})
	return s
}

// WithUser declares a regular (non-admin) user.
// Name is used as username, display name, and email prefix, and as the lookup
// key in SeedResult.User(name).
func (s *Seeder) WithUser(f UserFixture) *Seeder {
	s.t.Helper()
	if f.Name == "" {
		s.t.Fatalf("Seeder.WithUser: Name is required")
	}
	s.users = append(s.users, f)
	return s
}

// SetUserAccess records the host-group access for the named user.
// When bypass is true, the user bypasses the host check entirely.
// User and group names must match preceding declarations.
func (s *Seeder) SetUserAccess(user string, bypass bool, groups ...string) *Seeder {
	s.userAccesses = append(s.userAccesses, userAccessSpec{user: user, groups: groups, bypass: bypass})
	return s
}

// WithDevice declares a device owned by the named user.
// Name and OwnerUser are required. OwnerUser must match a preceding WithUser call.
func (s *Seeder) WithDevice(f DeviceFixture) *Seeder {
	s.t.Helper()
	if f.Name == "" {
		s.t.Fatalf("Seeder.WithDevice: Name is required")
	}
	if f.OwnerUser == "" {
		s.t.Fatalf("Seeder.WithDevice: OwnerUser is required (name=%q)", f.Name)
	}
	s.devices = append(s.devices, f)
	return s
}

// WithDeviceLeaseRule declares an address-lease rule to enable on a device after it is created.
// Device must match the Name of a preceding WithDevice call.
func (s *Seeder) WithDeviceLeaseRule(f DeviceLeaseRuleFixture) *Seeder {
	s.t.Helper()
	if f.Device == "" {
		s.t.Fatalf("Seeder.WithDeviceLeaseRule: Device is required")
	}
	if f.TTLSeconds <= 0 {
		s.t.Fatalf("Seeder.WithDeviceLeaseRule: TTLSeconds must be positive (device=%q)", f.Device)
	}
	s.leaseRules = append(s.leaseRules, f)
	return s
}

// WithDeviceMaxActiveRule declares a max-active-addresses rule to enable on a device after it is created.
// Device must match the Name of a preceding WithDevice call.
func (s *Seeder) WithDeviceMaxActiveRule(f DeviceMaxActiveRuleFixture) *Seeder {
	s.t.Helper()
	if f.Device == "" {
		s.t.Fatalf("Seeder.WithDeviceMaxActiveRule: Device is required")
	}
	if f.MaxAddresses <= 0 {
		s.t.Fatalf("Seeder.WithDeviceMaxActiveRule: MaxAddresses must be positive (device=%q)", f.Device)
	}
	s.maxActiveRules = append(s.maxActiveRules, f)
	return s
}

// WithAddress declares an IP address to register for the named device.
// Device and IP are required. Device must match a preceding WithDevice call.
func (s *Seeder) WithAddress(f AddressFixture) *Seeder {
	s.t.Helper()
	if f.Device == "" {
		s.t.Fatalf("Seeder.WithAddress: Device is required")
	}
	if f.IP == "" {
		s.t.Fatalf("Seeder.WithAddress: IP is required (device=%q)", f.Device)
	}
	s.addresses = append(s.addresses, f)
	return s
}

// WithAccessLogEntry declares one policy decision event to insert into the access log.
// ClientIP is required.  Devices and PolicyName are mutually exclusive.
func (s *Seeder) WithAccessLogEntry(f AccessLogEntryFixture) *Seeder {
	s.t.Helper()
	if f.ClientIP == "" {
		s.t.Fatalf("Seeder.WithAccessLogEntry: ClientIP is required")
	}
	if len(f.Devices) > 0 && f.PolicyName != "" {
		s.t.Fatalf("Seeder.WithAccessLogEntry: Devices and PolicyName are mutually exclusive (clientIP=%q)", f.ClientIP)
	}
	s.accessLogEntries = append(s.accessLogEntries, f)
	return s
}

// WithPolicyInitialize instructs Build to call PolicyService.Initialize after
// registering addresses, loading all enabled addresses into the in-memory cache.
// Required whenever a test needs the policy engine to reflect seeded addresses.
func (s *Seeder) WithPolicyInitialize() *Seeder {
	s.initPolicy = true
	return s
}

// Build materialises all declared entities in dependency order:
//
//  1. Create users
//  2. Reconcile groups (single call; Color and Icon default when empty)
//  3. Reconcile hosts with group memberships (single call)
//  4. Create network policies
//  5. Apply policy–group assignments
//  6. Apply user–group access rules
//  7. Create devices
//  8. Register device addresses
//  9. Initialize policy cache (if WithPolicyInitialize was called)
//
// 10. Insert access log entries
//
// Any failure calls t.Fatalf, so the test stops immediately.
func (s *Seeder) Build() *SeedResult {
	s.t.Helper()
	ctx := s.t.Context()

	result := &SeedResult{
		t:         s.t,
		policies:  make(map[string]ids.NetworkPolicyID, len(s.policies)),
		groups:    make(map[string]ids.HostGroupID, len(s.groups)),
		hosts:     make(map[string]ids.HostID, len(s.hosts)),
		users:     make(map[string]ids.UserID, len(s.users)),
		devices:   make(map[string]ids.DeviceID, len(s.devices)),
		addresses: make(map[string]ids.AddressID, len(s.addresses)),
	}

	// Build device→owner map used later for access log contributor resolution.
	deviceOwnerByName := make(map[string]string, len(s.devices))
	for _, d := range s.devices {
		deviceOwnerByName[d.Name] = d.OwnerUser
	}

	// 1. Users — CreateUser requires a super-admin principal
	if len(s.users) > 0 {
		adminPrincipal := AdminPrincipal(s.t, s.srv)
		for _, f := range s.users {
			u, err := s.srv.AuthService.CreateUser(ctx, f.Name, f.Name, f.Name+"@test.local", adminPrincipal)
			if err != nil {
				s.t.Fatalf("Seeder: create user %q: %v", f.Name, err)
			}
			result.users[f.Name] = u.ID
		}
	}

	// 2. Groups (single reconcile call)
	if len(s.groups) > 0 {
		desired := make([]hosts.DesiredHostGroup, len(s.groups))
		for i, f := range s.groups {
			color := f.Color
			if color == "" {
				color = "#000000"
			}
			icon := f.Icon
			if icon == "" {
				icon = "server"
			}
			desired[i] = hosts.DesiredHostGroup{Name: f.Name, Color: color, Icon: icon}
		}
		if err := s.srv.HostsService.ReconcileHostGroups(ctx, hosts.ReconcileHostGroupsInput{
			Groups: desired,
		}); err != nil {
			s.t.Fatalf("Seeder: reconcile groups: %v", err)
		}
		all, err := s.srv.HostsService.ListHostGroups(ctx)
		if err != nil {
			s.t.Fatalf("Seeder: list groups: %v", err)
		}
		for _, g := range all {
			result.groups[g.Name] = g.ID
		}
	}

	// 3. Hosts (single reconcile call, group names resolved to IDs)
	if len(s.hosts) > 0 {
		desired := make([]hosts.DesiredHost, len(s.hosts))
		for i, f := range s.hosts {
			groupIDs := make([]ids.HostGroupID, 0, len(f.Groups))
			for _, gname := range f.Groups {
				gid, ok := result.groups[gname]
				if !ok {
					s.t.Fatalf("Seeder: host %q references unknown group %q", f.FQDN, gname)
				}
				groupIDs = append(groupIDs, gid)
			}
			desired[i] = hosts.DesiredHost{FQDN: f.FQDN, GroupIDs: groupIDs}
		}
		if err := s.srv.HostsService.ReconcileHosts(ctx, hosts.ReconcileHostsInput{
			Hosts: desired,
		}); err != nil {
			s.t.Fatalf("Seeder: reconcile hosts: %v", err)
		}
		all, err := s.srv.HostsService.ListHosts(ctx)
		if err != nil {
			s.t.Fatalf("Seeder: list hosts: %v", err)
		}
		for _, h := range all {
			result.hosts[h.FQDN] = h.ID
		}
	}

	// 4. Policies
	for _, f := range s.policies {
		var desc *string
		if f.Desc != "" {
			d := f.Desc
			desc = &d
		}
		p, err := s.srv.NetworkPoliciesService.CreatePolicy(ctx, f.Name, f.CIDR, desc)
		if err != nil {
			s.t.Fatalf("Seeder: create policy %q: %v", f.Name, err)
		}
		result.policies[f.Name] = p.ID
	}

	// 5. Policy–group assignments
	for _, a := range s.assignments {
		policyID, ok := result.policies[a.policy]
		if !ok {
			s.t.Fatalf("Seeder: assignment references unknown policy %q", a.policy)
		}
		groupIDs := make([]ids.HostGroupID, 0, len(a.groups))
		for _, gname := range a.groups {
			gid, ok := result.groups[gname]
			if !ok {
				s.t.Fatalf("Seeder: assignment references unknown group %q", gname)
			}
			groupIDs = append(groupIDs, gid)
		}
		if err := s.srv.NetworkPoliciesService.SetHostAccess(ctx, policyID, false, groupIDs); err != nil {
			s.t.Fatalf("Seeder: assign groups to policy %q: %v", a.policy, err)
		}
	}

	// 6. User–group access rules
	for _, a := range s.userAccesses {
		userID, ok := result.users[a.user]
		if !ok {
			s.t.Fatalf("Seeder: user access references unknown user %q", a.user)
		}
		groupIDs := make([]ids.HostGroupID, 0, len(a.groups))
		for _, gname := range a.groups {
			gid, ok := result.groups[gname]
			if !ok {
				s.t.Fatalf("Seeder: user access references unknown group %q", gname)
			}
			groupIDs = append(groupIDs, gid)
		}
		if err := s.srv.UserAccessService.SetUserAccess(ctx, userID, a.bypass, groupIDs); err != nil {
			s.t.Fatalf("Seeder: set access for user %q: %v", a.user, err)
		}
	}

	// 7. Devices
	for _, f := range s.devices {
		ownerID, ok := result.users[f.OwnerUser]
		if !ok {
			s.t.Fatalf("Seeder: device %q references unknown user %q", f.Name, f.OwnerUser)
		}
		deviceID, _, err := s.srv.DeviceService.CreateDeviceWithAPIKey(ctx, f.Name, ownerID)
		if err != nil {
			s.t.Fatalf("Seeder: create device %q: %v", f.Name, err)
		}
		result.devices[f.Name] = deviceID
	}

	// 7b. Device rules (applied after devices, before addresses)
	for _, f := range s.leaseRules {
		deviceID, ok := result.devices[f.Device]
		if !ok {
			s.t.Fatalf("Seeder: lease rule references unknown device %q", f.Device)
		}
		if _, err := s.srv.RuleService.EnableDeviceAddressLeaseRule(ctx, deviceID, f.TTLSeconds); err != nil {
			s.t.Fatalf("Seeder: enable lease rule for device %q: %v", f.Device, err)
		}
	}
	for _, f := range s.maxActiveRules {
		deviceID, ok := result.devices[f.Device]
		if !ok {
			s.t.Fatalf("Seeder: max-active rule references unknown device %q", f.Device)
		}
		if _, err := s.srv.RuleService.EnableMaxActiveAddressesRule(ctx, deviceID, f.MaxAddresses); err != nil {
			s.t.Fatalf("Seeder: enable max-active rule for device %q: %v", f.Device, err)
		}
	}

	// 8. Device addresses
	var leaseRepo *lease.Repository // created lazily on first ExpiresAt use
	for _, f := range s.addresses {
		deviceID, ok := result.devices[f.Device]
		if !ok {
			s.t.Fatalf("Seeder: address %q references unknown device %q", f.IP, f.Device)
		}
		addr, _, err := s.srv.DeviceService.RegisterAddressActivity(ctx, deviceID, f.IP, device.EventSourceManual)
		if err != nil {
			s.t.Fatalf("Seeder: register address %q for device %q: %v", f.IP, f.Device, err)
		}
		result.addresses[addressKey(f.Device, f.IP)] = addr.ID

		if f.Disabled {
			if _, err := s.srv.DeviceService.DisableAddress(ctx, deviceID, addr.ID); err != nil {
				s.t.Fatalf("Seeder: disable address %q for device %q: %v", f.IP, f.Device, err)
			}
		}
		if f.ExpiresAt != nil {
			if leaseRepo == nil {
				leaseRepo = lease.NewRepository(s.srv.Database.DB())
			}
			if _, err := leaseRepo.UpsertAddressLease(ctx, &lease.AddressLease{
				AddressID: addr.ID,
				DeviceID:  deviceID,
				ExpiresAt: f.ExpiresAt,
			}); err != nil {
				s.t.Fatalf("Seeder: upsert lease for address %q device %q: %v", f.IP, f.Device, err)
			}
		}
	}

	// 9. Policy cache initialization
	if s.initPolicy {
		if err := s.srv.PolicyService.Initialize(ctx); err != nil {
			s.t.Fatalf("Seeder: initialize policy cache: %v", err)
		}
	}

	// 10. Access log entries
	if len(s.accessLogEntries) > 0 {
		events := make([]policy.DecisionEvent, 0, len(s.accessLogEntries))
		for _, f := range s.accessLogEntries {
			e := policy.DecisionEvent{
				ClientIP:   f.ClientIP,
				Outcome:    f.Outcome,
				DenyReason: f.DenyReason,
				CreatedAt:  time.Now().UTC(),
				Headers:    map[string][]string{},
				TargetHost: f.TargetHost,
			}
			switch {
			case f.PolicyName != "":
				policyID, ok := result.policies[f.PolicyName]
				if !ok {
					s.t.Fatalf("Seeder: access log entry references unknown policy %q", f.PolicyName)
				}
				e.MatchSource = policy.MatchSourceNetworkPolicy
				e.NetworkPolicyID = new(policyID)
				e.NetworkPolicyName = new(f.PolicyName)
			case len(f.Devices) > 0:
				contributors := make([]policy.IPContributor, 0, len(f.Devices))
				for _, devName := range f.Devices {
					deviceID, ok := result.devices[devName]
					if !ok {
						s.t.Fatalf("Seeder: access log entry references unknown device %q", devName)
					}
					addrID, ok := result.addresses[addressKey(devName, f.ClientIP)]
					if !ok {
						s.t.Fatalf("Seeder: access log entry for device %q: no address %q seeded — add WithAddress first", devName, f.ClientIP)
					}
					ownerName, ok := deviceOwnerByName[devName]
					if !ok {
						s.t.Fatalf("Seeder: access log entry: could not resolve owner for device %q", devName)
					}
					userID, ok := result.users[ownerName]
					if !ok {
						s.t.Fatalf("Seeder: access log entry: owner %q of device %q was not seeded", ownerName, devName)
					}
					contributors = append(contributors, policy.IPContributor{
						DeviceID: deviceID, AddressID: addrID, UserID: userID,
					})
				}
				e.IPContributors = contributors
				e.MatchSource = policy.MatchSourceDevice
			}
			events = append(events, e)
		}
		accessLogRepo := accesslog.NewRepository(s.srv.Database.DB())
		if err := accessLogRepo.BatchInsert(ctx, events); err != nil {
			s.t.Fatalf("Seeder: insert access log entries: %v", err)
		}
	}

	return result
}

// ── full-world preset ─────────────────────────────────────────────────────────

// SeedFullWorld returns a Seeder pre-loaded with a comprehensive, realistic
// dataset suitable for cross-domain query tests. Call Build() to materialise
// all entities; the returned *SeedResult lets tests look up any entity by name.
//
// Tests that need additional fixtures beyond the world preset can chain further
// With* calls before Build().
//
// Use the exported Fixture* variables to reference entity names and values in
// test assertions — they are the single source of truth for what this world contains.
//
// What is created:
//
//	Groups:      FixtureGroupEmpty, FixtureGroupBackend, FixtureGroupFrontend
//	Hosts:       FixtureHostBackend1+2 (backend); FixtureHostFrontend1+2 (frontend)
//	Users:       FixtureUserWithAccess (backend+frontend), FixtureUserNoAccess, FixtureUserBypassAccess (backend, bypass)
//	Policies:    FixturePolicyWithGroups (backend+frontend), FixturePolicyNoGroups (no groups)
//	Devices:     FixtureDeviceWithOwnerAccess (alice), FixtureDeviceWithoutOwnerAccess (bob), FixtureDeviceBypassAccess (charlie)
//	Rules:       FixtureLeaseRuleAliceLaptop (1h TTL), FixtureMaxActiveRuleAliceLaptop (max 2) — alice-laptop only
//	Addresses:   FixtureAddressAlice (10.1.0.1), FixtureAddressBob (10.2.0.1),
//	             FixtureAddressShared (charlie-desktop at 10.1.0.1 — shared with alice)
//	Policy cache: initialized; SharedIpCount=1 (10.1.0.1 owned by alice and charlie)
//	Access log:  FixtureAccessLogAliceAllow (allow, single contributor)
//	             FixtureAccessLogBobHostDeny (deny host_not_allowed, single contributor)
//	             FixtureAccessLogUnknownDeny (deny ip_not_registered, no contributor)
//	             FixtureAccessLogSharedIPAllow (allow, two contributors — shared IP path)
//	             FixtureAccessLogNetworkPolicyAllow (allow via network policy CIDR, no device contributors)
func SeedFullWorld(t *testing.T, srv *app.App) *Seeder {
	t.Helper()
	return NewSeeder(t, srv).
		WithGroup(FixtureGroupEmpty).
		WithGroup(FixtureGroupBackend).
		WithGroup(FixtureGroupFrontend).
		WithHost(FixtureHostBackend1).
		WithHost(FixtureHostBackend2).
		WithHost(FixtureHostFrontend1).
		WithHost(FixtureHostFrontend2).
		WithUser(FixtureUserWithAccess).
		WithUser(FixtureUserNoAccess).
		WithUser(FixtureUserBypassAccess).
		SetUserAccess(FixtureUserWithAccess.Name, false, FixtureGroupBackend.Name, FixtureGroupFrontend.Name).
		SetUserAccess(FixtureUserBypassAccess.Name, true, FixtureGroupBackend.Name).
		WithPolicy(FixturePolicyWithGroups).
		WithPolicy(FixturePolicyNoGroups).
		AssignGroupsToPolicy(FixturePolicyWithGroups.Name, FixtureGroupBackend.Name, FixtureGroupFrontend.Name).
		WithDevice(FixtureDeviceWithOwnerAccess).
		WithDevice(FixtureDeviceWithoutOwnerAccess).
		WithDevice(FixtureDeviceBypassAccess).
		WithDeviceLeaseRule(FixtureLeaseRuleAliceLaptop).
		WithDeviceMaxActiveRule(FixtureMaxActiveRuleAliceLaptop).
		WithAddress(FixtureAddressAlice).
		WithAddress(FixtureAddressBob).
		WithAddress(FixtureAddressShared).
		WithPolicyInitialize().
		WithAccessLogEntry(FixtureAccessLogAliceAllow).
		WithAccessLogEntry(FixtureAccessLogBobHostDeny).
		WithAccessLogEntry(FixtureAccessLogUnknownDeny).
		WithAccessLogEntry(FixtureAccessLogSharedIPAllow).
		WithAccessLogEntry(FixtureAccessLogNetworkPolicyAllow)
}
