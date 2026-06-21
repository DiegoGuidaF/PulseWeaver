//go:build test

package testutils

import (
	"fmt"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/devicepairing"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
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

// UserFixture describes the seeder inputs for a user.
// Name is the username; it also supplies the display name and email prefix
// (<name>@test.local) unless DisplayName / Email override them.
//
// Role selects the account's privilege level. The zero value seeds a plain
// user-role account (no password, exists only to own devices). AdminRole seeds
// a login-ready admin: the user is promoted via the real service path and its
// must_change_password flag is cleared, leaving SeededAdminPassword as a known,
// usable credential. SuperAdminRole is not seeded as a new row (there is no
// service path to a second superadmin) — reference the bootstrap admin instead
// via auth.BootstrapAdminUsername, which SeedResult.User resolves.
type UserFixture struct {
	Name string
	Role auth.Role
	// DisplayName overrides the rendered name when set; defaults to Name.
	DisplayName string
	// Email overrides the address when set; defaults to <Name>@test.local.
	Email string
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
	// Icon is the device's display icon (emoji or legacy Tabler name); optional.
	Icon string
	// GenerateAPIKey mints an API key so the device renders with a key prefix; optional.
	GenerateAPIKey bool
}

// AddressFixture describes the seeder inputs for a device address.
// Device must match the Name of a DeviceFixture seeded in the same Build call.
// ExpiresAt creates a lease for the address if set. Disabled disables the address after registration.
type AddressFixture struct {
	Device    string
	IP        string
	ExpiresAt *time.Time         // if set, a lease is created with this expiry
	Disabled  bool               // if true, the address is disabled after registration
	Source    device.EventSource // registration source; defaults to EventSourceManual
}

// PairingFixture describes a device pairing to seed.
// Device must match the Name of a DeviceFixture seeded in the same Build call.
// Status is one of: pending, used, invalidated, expired.
// Expired pairings are inserted via the repository directly with a past expiry
// since the service always sets a future expiry.
type PairingFixture struct {
	Device string
	Status string // "pending" | "used" | "invalidated" | "expired"
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
	TargetURI  *string  // optional
	HTTPMethod *string  // optional
	DurationUs int64    // optional; request processing duration in microseconds
	// GeoIP, when set, populates the access_log_geoip child row. Leave nil for
	// requests with no geolocation (country_code resolves to NULL).
	GeoIP *geoip.Result
}

// ── world fixture variables ───────────────────────────────────────────────────

// World fixture variables are named by their role in the test world, not by
// domain semantics. They are used by SeedFullWorld and by cross-domain query
// tests that need to assert against the seeded values without hardcoding strings.

var (
	FixturePolicyWithGroups      = PolicyFixture{Name: "corp-vpn", CIDR: "10.0.0.0/12", Desc: "Corporate VPN access"}
	FixturePolicyNoGroups        = PolicyFixture{Name: "isolated", CIDR: "172.16.0.0/12"}
	FixturePolicyBypassHostCheck = PolicyFixture{Name: "ops-network", CIDR: "192.168.0.0/16"}

	// The full world seeds GroupMedia + GroupProductivity (two hosts each) and
	// GroupInfrastructure (no hosts — the empty-group path); all three come from
	// the shared roster in seeder_identity.go.

	FixtureHostBackend1  = HostFixture{FQDN: "api1.internal", Groups: []string{GroupMedia.Name}}
	FixtureHostBackend2  = HostFixture{FQDN: "api2.internal", Groups: []string{GroupMedia.Name}}
	FixtureHostFrontend1 = HostFixture{FQDN: "web1.internal", Groups: []string{GroupProductivity.Name}}
	FixtureHostFrontend2 = HostFixture{FQDN: "web2.internal", Groups: []string{GroupProductivity.Name}}

	FixtureUserWithAccess   = PersonJames // Media + Productivity, no bypass
	FixtureUserNoAccess     = PersonNoah  // no group access
	FixtureUserBypassAccess = PersonMaria // Media with bypass=true

	FixtureDeviceWithOwnerAccess    = DeviceFixture{Name: "james-laptop", OwnerUser: FixtureUserWithAccess.Name}
	FixtureDeviceWithoutOwnerAccess = DeviceFixture{Name: "noah-phone", OwnerUser: FixtureUserNoAccess.Name}
	FixtureDeviceBypassAccess       = DeviceFixture{Name: "maria-desktop", OwnerUser: FixtureUserBypassAccess.Name}

	// Rules seeded on the with-access device (james-laptop) by SeedFullWorld.
	FixtureLeaseRuleAliceLaptop     = DeviceLeaseRuleFixture{Device: FixtureDeviceWithOwnerAccess.Name, TTLSeconds: 3600}
	FixtureMaxActiveRuleAliceLaptop = DeviceMaxActiveRuleFixture{Device: FixtureDeviceWithOwnerAccess.Name, MaxAddresses: 2}

	FixtureAddressAlice         = AddressFixture{Device: FixtureDeviceWithOwnerAccess.Name, IP: "10.1.0.1"}
	FixtureAddressAliceDisabled = AddressFixture{Device: FixtureDeviceWithOwnerAccess.Name, IP: "10.4.0.3", Disabled: true}
	FixtureAddressBob           = AddressFixture{Device: FixtureDeviceWithoutOwnerAccess.Name, IP: "10.2.0.1"}
	FixtureAddressShared        = AddressFixture{Device: FixtureDeviceBypassAccess.Name, IP: "10.1.0.1"} // Maria's device shares James's IP

	// Liam owns two devices seeded purely to exercise the pairing summary statuses.
	FixtureUserPairing          = PersonLiam
	FixtureDevicePairingUsed    = DeviceFixture{Name: "liam-used-device", OwnerUser: FixtureUserPairing.Name}
	FixtureDevicePairingExpired = DeviceFixture{Name: "liam-expired-device", OwnerUser: FixtureUserPairing.Name}

	// Pairing fixtures covering all four non-nil last_pairing statuses.
	FixturePairingBobPending         = PairingFixture{Device: FixtureDeviceWithoutOwnerAccess.Name, Status: "pending"}
	FixturePairingCharlieInvalidated = PairingFixture{Device: FixtureDeviceBypassAccess.Name, Status: "invalidated"}
	FixturePairingDianaUsed          = PairingFixture{Device: FixtureDevicePairingUsed.Name, Status: "used"}
	FixturePairingDianaExpired       = PairingFixture{Device: FixtureDevicePairingExpired.Name, Status: "expired"}

	// Six canonical access-log paths exercised by SeedFullWorld:
	// 1. allow — single contributor (device+user link)
	// 2. deny  — single contributor (host not allowed)
	// 3. deny  — no contributor (IP not registered)
	// 4. allow — multiple contributors for one entry (shared IP, two users)
	// 5. allow — network policy CIDR match (no device contributors)
	// 6. allow — bypass network policy CIDR match (no device contributors, any host)
	FixtureAccessLogAliceAllow         = AccessLogEntryFixture{ClientIP: "10.1.0.1", Outcome: true, Devices: []string{FixtureDeviceWithOwnerAccess.Name}, TargetHost: new(FixtureHostBackend1.FQDN)}
	FixtureAccessLogBobHostDeny        = AccessLogEntryFixture{ClientIP: "10.2.0.1", Outcome: false, DenyReason: new(policy.DenyReasonHostNotAllowed), Devices: []string{FixtureDeviceWithoutOwnerAccess.Name}, TargetHost: new(FixtureHostBackend2.FQDN)}
	FixtureAccessLogUnknownDeny        = AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: new(FixtureHostFrontend1.FQDN)}
	FixtureAccessLogSharedIPAllow      = AccessLogEntryFixture{ClientIP: "10.1.0.1", Outcome: true, Devices: []string{FixtureDeviceWithOwnerAccess.Name, FixtureDeviceBypassAccess.Name}, TargetHost: new(FixtureHostFrontend2.FQDN)}
	FixtureAccessLogNetworkPolicyAllow = AccessLogEntryFixture{ClientIP: "10.3.0.1", Outcome: true, PolicyName: FixturePolicyWithGroups.Name, TargetHost: new(FixtureHostBackend1.FQDN)}
	FixtureAccessLogBypassAllow        = AccessLogEntryFixture{ClientIP: "192.168.1.50", Outcome: true, PolicyName: FixturePolicyBypassHostCheck.Name, TargetHost: new(FixtureHostFrontend1.FQDN)}

	// Geolocated external traffic: denied (unregistered) requests from public IPs
	// carrying GeoIP, distinct durations, HTTP methods, and (some) target URIs.
	// These exercise the country/continent, http_method, target_uri, and
	// duration-sort filters; the six entries above have no GeoIP (country NULL),
	// so together they cover the is_null / NULL-inclusion paths too.
	FixtureGeoGermany = GeoResult("DE", 3320, "Deutsche Telekom")
	FixtureGeoUSA     = GeoResult("US", 15169, "Google LLC")
	FixtureGeoSpain   = GeoResult("ES", 12479, "Orange Espagne")

	FixtureAccessLogGeoGermanyAPI   = AccessLogEntryFixture{ClientIP: "198.51.100.10", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: new(FixtureHostBackend1.FQDN), TargetURI: new("/api/users"), HTTPMethod: new("GET"), DurationUs: 30, GeoIP: &FixtureGeoGermany}
	FixtureAccessLogGeoGermanyLogin = AccessLogEntryFixture{ClientIP: "198.51.100.11", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: new(FixtureHostBackend2.FQDN), TargetURI: new("/api/login"), HTTPMethod: new("POST"), DurationUs: 220, GeoIP: &FixtureGeoGermany}
	FixtureAccessLogGeoUSA          = AccessLogEntryFixture{ClientIP: "198.51.100.20", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: new(FixtureHostFrontend1.FQDN), HTTPMethod: new("GET"), DurationUs: 150, GeoIP: &FixtureGeoUSA}
	FixtureAccessLogGeoSpain        = AccessLogEntryFixture{ClientIP: "198.51.100.30", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: new(FixtureHostFrontend2.FQDN), HTTPMethod: new("DELETE"), DurationUs: 90, GeoIP: &FixtureGeoSpain}
)

// SeededAdminPassword is the known, login-ready password assigned to every
// AdminRole UserFixture (see UserFixture.Role). The bootstrap superadmin keeps
// TestPassword/TestAdminPassword. Both are exported so security audits and
// manual-testing scenarios can authenticate against a generated seed DB.
const SeededAdminPassword = TestAdminPassword

// ── PW-66 enrichment fixtures ─────────────────────────────────────────────────
//
// These broaden SeedFullWorld into a genuinely complete/complex DB: every role
// owns a device+address, plus edge/NULL entities and adversarial-string names.
// They serve both the cross-domain query tests and the PW-66 security audit.

var (
	// Role coverage: a login-ready admin (Sarah) and the bootstrap superadmin,
	// each owning a device with one address. The plain user role is already
	// covered by FixtureUserWithAccess (James).
	FixtureUserAdmin         = PersonSarah
	FixtureDeviceAdmin       = DeviceFixture{Name: "sarah-laptop", OwnerUser: FixtureUserAdmin.Name}
	FixtureAddressAdmin      = AddressFixture{Device: FixtureDeviceAdmin.Name, IP: "10.4.0.1"}
	FixtureDeviceSuperAdmin  = DeviceFixture{Name: "admin-laptop", OwnerUser: auth.BootstrapAdminUsername}
	FixtureAddressSuperAdmin = AddressFixture{Device: FixtureDeviceSuperAdmin.Name, IP: "10.5.0.1"}

	// Edge / NULL entities are owned by a dedicated user (Tom) so they do not
	// perturb the per-owner assertions of the James/Noah/Maria/Liam fixtures.
	// Tom has no group access.
	FixtureUserEdge = PersonTom
	// Orphan device: created with no address (no WithAddress call).
	FixtureDeviceOrphan = DeviceFixture{Name: "tom-orphan", OwnerUser: FixtureUserEdge.Name}
	// Device with multiple addresses.
	FixtureDeviceMultiAddr = DeviceFixture{Name: "tom-multi", OwnerUser: FixtureUserEdge.Name}
	FixtureAddressMultiA   = AddressFixture{Device: FixtureDeviceMultiAddr.Name, IP: "10.6.0.1"}
	FixtureAddressMultiB   = AddressFixture{Device: FixtureDeviceMultiAddr.Name, IP: "10.6.0.2"}
	// Disabled address.
	FixtureDeviceDisabledAddr = DeviceFixture{Name: "tom-disabled", OwnerUser: FixtureUserEdge.Name}
	FixtureAddressDisabled    = AddressFixture{Device: FixtureDeviceDisabledAddr.Name, IP: "10.7.0.1", Disabled: true}

	// Third contributor on the shared IP 10.1.0.1: james-laptop + maria-desktop
	// + priya-laptop. Priya is a bypass user (host check bypassed, no group
	// membership), so it contributes "all hosts" to the contributor intersection —
	// the shared-IP allow path is unchanged and Priya does not perturb the
	// per-group user-count assertions, while the IP now exercises the
	// 3+-contributor query/cache paths.
	FixtureUserSharedExtra    = PersonPriya
	FixtureDeviceSharedThird  = DeviceFixture{Name: "priya-laptop", OwnerUser: FixtureUserSharedExtra.Name}
	FixtureAddressSharedThird = AddressFixture{Device: FixtureDeviceSharedThird.Name, IP: "10.1.0.1"}

	// Adversarial-string names — escaping/encoding paths and the stored-XSS
	// render precondition for the audit. Only group names and policy
	// name/description carry these: host FQDNs are rejected by hosts.ValidateFQDN.
	FixtureGroupAdversarial  = GroupFixture{Name: `<script>alert('xss')</script>`}
	FixturePolicyAdversarial = PolicyFixture{
		Name: `"; DROP TABLE policies;-- ☠ 测试`,
		CIDR: "10.99.0.0/16",
		Desc: `<img src=x onerror=alert(1)>`,
	}
)

// ── IPv6 fixtures (seed-generator only) ───────────────────────────────────────
//
// Kept out of SeedFullWorld so it does not shift the entity counts the
// cross-domain query tests assert. The seed-DB generator chains these on so the
// generated artifact exercises IPv6 grants and addresses.
var (
	// IPv6 site allocation in the "normal" band (/48 is narrower than the /47 warn
	// line), so it is accepted without the broad-CIDR flag.
	FixturePolicyIPv6 = PolicyFixture{Name: "home-ipv6", CIDR: "2001:db8::/48", Desc: "IPv6 home LAN"}
	// An IPv6 address making the multi-address device (tom-multi) dual-stack, for IPv6 rendering.
	FixtureAddressIPv6 = AddressFixture{Device: FixtureDeviceMultiAddr.Name, IP: "2001:db8::1"}
)

// ── relational spec types (internal) ─────────────────────────────────────────

type policyAccessSpec struct {
	policy string
	groups []string
	bypass bool
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
	t *testing.T

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
	pairings         []PairingFixture
	accessLogEntries []AccessLogEntryFixture
	accessLogVolume  int
	observedHosts    []observedHostSpec
	trafficProfile   *TrafficProfile
	initPolicy       bool
}

// observedHostSpec records an FQDN that should appear in the access log as
// observed traffic (denied, no-contributor rows), surfacing it as a host
// suggestion. count controls how many rows, so suggestion ordering can be exercised.
type observedHostSpec struct {
	fqdn  string
	count int
}

// NewSeeder returns a fresh Seeder. Call With* methods to declare fixtures, then
// Build(srv) to materialise them against a specific app instance.
func NewSeeder(t *testing.T) *Seeder {
	t.Helper()
	return &Seeder{t: t}
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

// WithPolicyBypassHostCheck records that the named policy should have bypass_host_check=true
// with no group restrictions. Policy name must match a preceding WithPolicy call.
func (s *Seeder) WithPolicyBypassHostCheck(policy string) *Seeder {
	s.assignments = append(s.assignments, policyAccessSpec{policy: policy, bypass: true})
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

// WithPairing declares a device pairing to seed after devices are created.
// Device must match the Name of a preceding WithDevice call.
// Status must be one of: pending, used, invalidated, expired.
func (s *Seeder) WithPairing(f PairingFixture) *Seeder {
	s.t.Helper()
	if f.Device == "" {
		s.t.Fatalf("Seeder.WithPairing: Device is required")
	}
	if f.Status == "" {
		s.t.Fatalf("Seeder.WithPairing: Status is required (device=%q)", f.Device)
	}
	s.pairings = append(s.pairings, f)
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

// WithAccessLogVolume instructs Build to insert n additional synthetic access-log
// rows (denied, no contributors, spread over distinct client IPs) after the
// explicit WithAccessLogEntry rows. This is an opt-in volume builder — it is NOT
// part of SeedFullWorld, so it does not slow every integration test that
// materialises the world. Pagination cases and the seed-DB generator opt in.
func (s *Seeder) WithAccessLogVolume(n int) *Seeder {
	s.t.Helper()
	if n < 0 {
		s.t.Fatalf("Seeder.WithAccessLogVolume: n must be non-negative (got %d)", n)
	}
	s.accessLogVolume = n
	return s
}

// WithObservedHost appends `count` denied, no-contributor access-log rows whose
// target_host is fqdn, spread over distinct client IPs. Because host suggestions
// are derived from access_log.target_host (for FQDNs that are neither a known
// host nor ignored), this surfaces fqdn as a suggestion; varying count across
// calls exercises the frequency-based ordering.
func (s *Seeder) WithObservedHost(fqdn string, count int) *Seeder {
	s.t.Helper()
	if fqdn == "" {
		s.t.Fatalf("Seeder.WithObservedHost: fqdn is required")
	}
	if count <= 0 {
		s.t.Fatalf("Seeder.WithObservedHost: count must be positive (fqdn=%q)", fqdn)
	}
	s.observedHosts = append(s.observedHosts, observedHostSpec{fqdn: fqdn, count: count})
	return s
}

// WithPolicyInitialize instructs Build to call PolicyService.Initialize after
// registering addresses, loading all enabled addresses into the in-memory cache.
// Required whenever a test needs the policy engine to reflect seeded addresses.
func (s *Seeder) WithPolicyInitialize() *Seeder {
	s.initPolicy = true
	return s
}

// referencesBootstrapAdmin reports whether any declared device is owned by the
// bootstrap admin, so Build knows to register that account even when no regular
// users are seeded.
func (s *Seeder) referencesBootstrapAdmin() bool {
	for _, d := range s.devices {
		if d.OwnerUser == auth.BootstrapAdminUsername {
			return true
		}
	}
	return false
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
func (s *Seeder) Build(srv *app.App) *SeedResult {
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

	// 1. Users — CreateUser requires a super-admin principal. The bootstrap admin
	// (itself a superadmin) is registered under its username so fixtures can own
	// devices as the superadmin and SeedResult.User(auth.BootstrapAdminUsername)
	// resolves it.
	if len(s.users) > 0 || s.referencesBootstrapAdmin() {
		adminPrincipal := AdminPrincipal(s.t, srv)
		result.users[auth.BootstrapAdminUsername] = adminPrincipal.UserID
		for _, f := range s.users {
			displayName := f.DisplayName
			if displayName == "" {
				displayName = f.Name
			}
			email := f.Email
			if email == "" {
				email = f.Name + "@test.local"
			}
			u, err := srv.AuthService.CreateUser(ctx, f.Name, displayName, new(email), adminPrincipal)
			if err != nil {
				s.t.Fatalf("Seeder: create user %q: %v", f.Name, err)
			}
			result.users[f.Name] = u.ID

			switch f.Role {
			case "", auth.UserRole:
				// Plain user-role account (no password); nothing further.
			case auth.AdminRole:
				// Promote via the real service path, then clear must_change_password
				// through ChangePassword so the admin is login-ready with the known
				// SeededAdminPassword.
				if _, err := srv.AuthService.PromoteUser(ctx, adminPrincipal, u.ID, SeededAdminPassword); err != nil {
					s.t.Fatalf("Seeder: promote user %q to admin: %v", f.Name, err)
				}
				if err := srv.AuthService.ChangePassword(ctx, u.ID, ids.SessionID(0), SeededAdminPassword, SeededAdminPassword); err != nil {
					s.t.Fatalf("Seeder: clear must_change_password for %q: %v", f.Name, err)
				}
			case auth.SuperAdminRole:
				s.t.Fatalf("Seeder: UserFixture %q requests SuperAdminRole, which has no seed path; reference auth.BootstrapAdminUsername instead", f.Name)
			default:
				s.t.Fatalf("Seeder: UserFixture %q has unknown role %q", f.Name, f.Role)
			}
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
		if err := srv.HostsService.ReconcileHostGroups(ctx, hosts.ReconcileHostGroupsInput{
			Groups: desired,
		}); err != nil {
			s.t.Fatalf("Seeder: reconcile groups: %v", err)
		}
		all, err := srv.HostsService.ListHostGroups(ctx)
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
		if err := srv.HostsService.ReconcileHosts(ctx, hosts.ReconcileHostsInput{
			Hosts: desired,
		}); err != nil {
			s.t.Fatalf("Seeder: reconcile hosts: %v", err)
		}
		all, err := srv.HostsService.ListHosts(ctx)
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
		p, err := srv.NetworkPoliciesService.CreatePolicy(ctx, f.Name, f.CIDR, desc)
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
		if err := srv.NetworkPoliciesService.SetHostAccess(ctx, policyID, a.bypass, groupIDs); err != nil {
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
		if err := srv.UserAccessService.SetUserAccess(ctx, userID, a.bypass, groupIDs); err != nil {
			s.t.Fatalf("Seeder: set access for user %q: %v", a.user, err)
		}
	}

	// 7. Devices
	for _, f := range s.devices {
		ownerID, ok := result.users[f.OwnerUser]
		if !ok {
			s.t.Fatalf("Seeder: device %q references unknown user %q", f.Name, f.OwnerUser)
		}
		input := device.CreateDeviceInput{Name: f.Name, GenerateAPIKey: f.GenerateAPIKey}
		if f.Icon != "" {
			input.Icon = new(f.Icon)
		}
		dev, _, err := srv.DeviceService.CreateDeviceWithOptions(ctx, &auth.Principal{UserID: ownerID}, input)
		if err != nil {
			s.t.Fatalf("Seeder: create device %q: %v", f.Name, err)
		}
		result.devices[f.Name] = dev.ID
	}

	// 7b. Device rules (applied after devices, before addresses)
	for _, f := range s.leaseRules {
		deviceID, ok := result.devices[f.Device]
		if !ok {
			s.t.Fatalf("Seeder: lease rule references unknown device %q", f.Device)
		}
		if _, err := srv.RuleService.EnableDeviceAddressLeaseRule(ctx, deviceID, f.TTLSeconds); err != nil {
			s.t.Fatalf("Seeder: enable lease rule for device %q: %v", f.Device, err)
		}
	}
	for _, f := range s.maxActiveRules {
		deviceID, ok := result.devices[f.Device]
		if !ok {
			s.t.Fatalf("Seeder: max-active rule references unknown device %q", f.Device)
		}
		if _, err := srv.RuleService.EnableMaxActiveAddressesRule(ctx, deviceID, f.MaxAddresses); err != nil {
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
		source := f.Source
		if source == "" {
			source = device.EventSourceManual
		}
		addr, _, err := srv.DeviceService.RegisterAddressActivity(ctx, deviceID, f.IP, source)
		if err != nil {
			s.t.Fatalf("Seeder: register address %q for device %q: %v", f.IP, f.Device, err)
		}
		result.addresses[addressKey(f.Device, f.IP)] = addr.ID

		if f.Disabled {
			if _, err := srv.DeviceService.DisableAddress(ctx, deviceID, addr.ID); err != nil {
				s.t.Fatalf("Seeder: disable address %q for device %q: %v", f.IP, f.Device, err)
			}
		}
		if f.ExpiresAt != nil {
			if leaseRepo == nil {
				leaseRepo = lease.NewRepository(srv.Database.DB())
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

	// 8b. Device pairings
	if len(s.pairings) > 0 {
		pairingRepo := devicepairing.NewRepository(srv.Database.DB())
		for _, f := range s.pairings {
			deviceID, ok := result.devices[f.Device]
			if !ok {
				s.t.Fatalf("Seeder: pairing references unknown device %q", f.Device)
			}
			switch f.Status {
			case "pending":
				_, err := srv.DevicePairingService.CreatePairing(ctx, devicepairing.CreatePairingRequest{
					DeviceID: deviceID, HeartbeatServerURL: "https://pulse.example.com",
					IntervalSeconds: 900, ExpiresInHours: 24,
				})
				if err != nil {
					s.t.Fatalf("Seeder: create pending pairing for device %q: %v", f.Device, err)
				}
			case "used":
				pairing, err := srv.DevicePairingService.CreatePairing(ctx, devicepairing.CreatePairingRequest{
					DeviceID: deviceID, HeartbeatServerURL: "https://pulse.example.com",
					IntervalSeconds: 900, ExpiresInHours: 24,
				})
				if err != nil {
					s.t.Fatalf("Seeder: create pairing for used device %q: %v", f.Device, err)
				}
				if _, err := srv.DevicePairingService.ClaimPairing(ctx, pairing.PairingCode); err != nil {
					s.t.Fatalf("Seeder: claim pairing for device %q: %v", f.Device, err)
				}
			case "invalidated":
				pairing, err := srv.DevicePairingService.CreatePairing(ctx, devicepairing.CreatePairingRequest{
					DeviceID: deviceID, HeartbeatServerURL: "https://pulse.example.com",
					IntervalSeconds: 900, ExpiresInHours: 24,
				})
				if err != nil {
					s.t.Fatalf("Seeder: create pairing for invalidated device %q: %v", f.Device, err)
				}
				if err := srv.DevicePairingService.InvalidatePairing(ctx, deviceID, pairing.ID); err != nil {
					s.t.Fatalf("Seeder: invalidate pairing for device %q: %v", f.Device, err)
				}
			case "expired":
				// Service always sets a future expiry so we bypass it and insert via the repo directly.
				_, err := pairingRepo.CreatePairing(ctx, devicepairing.CreatePairingRequest{
					DeviceID:           deviceID,
					PairingCode:        fmt.Sprintf("test-expired-pairing-%d", deviceID),
					HeartbeatServerURL: "https://pulse.example.com",
					IntervalSeconds:    900,
					ExpiresAt:          time.Now().Add(-1 * time.Hour),
				})
				if err != nil {
					s.t.Fatalf("Seeder: create expired pairing for device %q: %v", f.Device, err)
				}
			default:
				s.t.Fatalf("Seeder.WithPairing: unknown status %q for device %q", f.Status, f.Device)
			}
		}
	}

	// 9. Policy cache initialization
	if s.initPolicy {
		if err := srv.PolicyService.Initialize(ctx); err != nil {
			s.t.Fatalf("Seeder: initialize policy cache: %v", err)
		}
	}

	// 10. Access log entries (explicit fixtures + opt-in synthetic volume + generated traffic)
	if len(s.accessLogEntries) > 0 || s.accessLogVolume > 0 || len(s.observedHosts) > 0 || s.trafficProfile != nil {
		events := make([]policy.DecisionEvent, 0, len(s.accessLogEntries)+s.accessLogVolume)
		for _, f := range s.accessLogEntries {
			events = append(events, s.buildDecisionEvent(f, time.Now().UTC(), result, deviceOwnerByName))
		}
		// Generated traffic: a weighted, time-distributed set of events whose
		// timestamps spread across the profile window ending now, so dashboard
		// time-series widgets show a curve rather than a single spike.
		if s.trafficProfile != nil {
			events = append(events, s.generateTraffic(*s.trafficProfile, result, deviceOwnerByName)...)
		}
		// Synthetic volume: cheap denied, no-contributor rows over distinct IPs.
		for i := 0; i < s.accessLogVolume; i++ {
			events = append(events, policy.DecisionEvent{
				ClientIP:   fmt.Sprintf("100.64.%d.%d", (i/256)%256, i%256),
				Outcome:    false,
				DenyReason: new(policy.DenyReasonIPNotRegistered),
				CreatedAt:  time.Now().UTC(),
				Headers:    map[string][]string{},
			})
		}
		// Observed-but-unknown hosts: denied, no-contributor rows carrying a
		// target_host, which surfaces the FQDN as a host suggestion. Distinct
		// client IPs in TEST-NET-3 (203.0.113.0/24) keep rows independent.
		obsIP := 1
		for _, oh := range s.observedHosts {
			for i := 0; i < oh.count; i++ {
				host := oh.fqdn
				events = append(events, policy.DecisionEvent{
					ClientIP:   fmt.Sprintf("203.0.113.%d", obsIP),
					Outcome:    false,
					DenyReason: new(policy.DenyReasonIPNotRegistered),
					CreatedAt:  time.Now().UTC(),
					Headers:    map[string][]string{},
					TargetHost: &host,
				})
				obsIP++
			}
		}
		accessLogRepo := accesslog.NewRepository(srv.Database.DB())
		if err := accessLogRepo.BatchInsert(ctx, events); err != nil {
			s.t.Fatalf("Seeder: insert access log entries: %v", err)
		}
	}

	return result
}

// ── composable add-on presets ─────────────────────────────────────────────────

// WithAdversarialEntities chains the adversarial-string group and policy — a
// <script> group name plus a SQL-injection policy name with an <img onerror>
// description. They are the escaping/encoding and stored-XSS render precondition
// for the security audit. Self-contained, so any world (including the sample one)
// can layer them on when a test or audit needs an injection target; kept out of
// the sample preset itself so the hostile strings never reach a screenshot.
func (s *Seeder) WithAdversarialEntities() *Seeder {
	s.t.Helper()
	return s.
		WithGroup(FixtureGroupAdversarial).
		WithPolicy(FixturePolicyAdversarial)
}

// WithEdgeEntities chains the NULL/edge device shapes owned by FixtureUserEdge
// (grace, no group access): an orphan device with no address, a device with two
// addresses, and a device with a single disabled address. Self-contained, so any
// base world can layer it on to exercise the orphan / multi-address /
// disabled-address query and rendering paths.
func (s *Seeder) WithEdgeEntities() *Seeder {
	s.t.Helper()
	return s.
		WithUser(FixtureUserEdge).
		WithDevice(FixtureDeviceOrphan).
		WithDevice(FixtureDeviceMultiAddr).
		WithDevice(FixtureDeviceDisabledAddr).
		WithAddress(FixtureAddressMultiA).
		WithAddress(FixtureAddressMultiB).
		WithAddress(FixtureAddressDisabled)
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
//	Identities come from the shared roster (see seeder_identity.go): the
//	role-named Fixture* variables back onto real people — FixtureUserWithAccess is
//	James, FixtureUserNoAccess is Noah, FixtureUserBypassAccess is Maria,
//	FixtureUserPairing is Liam, FixtureUserAdmin is Sarah, FixtureUserEdge is Tom,
//	FixtureUserSharedExtra is Priya — and the two real service groups (Media,
//	Productivity). Addresses, CIDRs and the access log stay test-private here.
//
//	Groups:      GroupInfrastructure (empty — no hosts), GroupMedia, GroupProductivity,
//	             FixtureGroupAdversarial (<script> name — escaping/XSS-render precondition)
//	Hosts:       FixtureHostBackend1+2 (Media); FixtureHostFrontend1+2 (Productivity)
//	Users:       FixtureUserWithAccess (James, Media+Productivity), FixtureUserNoAccess (Noah),
//	             FixtureUserBypassAccess (Maria, Media bypass), FixtureUserPairing (Liam),
//	             FixtureUserAdmin (Sarah, admin role, login-ready via SeededAdminPassword),
//	             FixtureUserEdge (Tom, owns the NULL/edge devices, no access),
//	             FixtureUserSharedExtra (Priya, bypass user, third contributor on 10.1.0.1);
//	             the bootstrap admin (auth.BootstrapAdminUsername) is the superadmin and is
//	             registered in SeedResult so it can own devices
//	Policies:    FixturePolicyWithGroups (Media+Productivity), FixturePolicyNoGroups (no groups),
//	             FixturePolicyBypassHostCheck (ops-network, bypass_host_check=true, 192.168.0.0/16),
//	             FixturePolicyAdversarial (quote/unicode name + <img onerror> desc, 10.99.0.0/16)
//	Devices:     FixtureDeviceWithOwnerAccess (james-laptop), FixtureDeviceWithoutOwnerAccess (noah-phone),
//	             FixtureDeviceBypassAccess (maria-desktop),
//	             FixtureDevicePairingUsed (liam-used-device), FixtureDevicePairingExpired (liam-expired-device),
//	             FixtureDeviceAdmin (sarah-laptop), FixtureDeviceSuperAdmin (admin-laptop, owned by bootstrap admin),
//	             FixtureDeviceOrphan (tom-orphan, no address), FixtureDeviceMultiAddr (tom-multi, two addresses),
//	             FixtureDeviceSharedThird (priya-laptop, third contributor on 10.1.0.1),
//	             FixtureDeviceDisabledAddr (tom-disabled, one disabled address)
//	Rules:       FixtureLeaseRuleAliceLaptop (1h TTL), FixtureMaxActiveRuleAliceLaptop (max 2) — james-laptop only
//	Addresses:   FixtureAddressAlice (10.1.0.1), FixtureAddressBob (10.2.0.1),
//	             FixtureAddressShared (maria-desktop at 10.1.0.1 — shared with james-laptop),
//	             FixtureAddressAdmin (Sarah 10.4.0.1), FixtureAddressSuperAdmin (admin 10.5.0.1),
//	             FixtureAddressMultiA/B (tom-multi 10.6.0.1/2), FixtureAddressSharedThird (priya-laptop 10.1.0.1),
//	             FixtureAddressDisabled (tom-disabled 10.7.0.1, disabled)
//	Pairings:    FixturePairingBobPending (noah-phone, pending),
//	             FixturePairingCharlieInvalidated (maria-desktop, invalidated),
//	             FixturePairingDianaUsed (liam-used-device, used),
//	             FixturePairingDianaExpired (liam-expired-device, expired)
//	             james-laptop has no pairing (nil last_pairing)
//	Policy cache: initialized; SharedIpCount=1 (10.1.0.1 owned by James, Maria and Priya)
//	Access log:  FixtureAccessLogAliceAllow (allow, single contributor)
//	             FixtureAccessLogBobHostDeny (deny host_not_allowed, single contributor)
//	             FixtureAccessLogUnknownDeny (deny ip_not_registered, no contributor)
//	             FixtureAccessLogSharedIPAllow (allow, two contributors — shared IP path, ambiguous)
//	             FixtureAccessLogNetworkPolicyAllow (allow via network policy CIDR, no device contributors)
//	             FixtureAccessLogBypassAllow (allow via bypass CIDR, no device contributors)
//	             FixtureAccessLogGeoGermanyAPI/Login, GeoUSA, GeoSpain (geolocated external
//	               traffic with GeoIP, distinct durations/methods/URIs — the six above have
//	               no GeoIP, so country_code is NULL on them)
func SeedFullWorld(t *testing.T) *Seeder {
	t.Helper()
	return NewSeeder(t).
		WithGroup(GroupInfrastructure).
		WithGroup(GroupMedia).
		WithGroup(GroupProductivity).
		WithHost(FixtureHostBackend1).
		WithHost(FixtureHostBackend2).
		WithHost(FixtureHostFrontend1).
		WithHost(FixtureHostFrontend2).
		WithUser(FixtureUserWithAccess).
		WithUser(FixtureUserNoAccess).
		WithUser(FixtureUserBypassAccess).
		SetUserAccess(FixtureUserWithAccess.Name, false, GroupMedia.Name, GroupProductivity.Name).
		SetUserAccess(FixtureUserBypassAccess.Name, true, GroupMedia.Name).
		WithPolicy(FixturePolicyWithGroups).
		WithPolicy(FixturePolicyNoGroups).
		WithPolicy(FixturePolicyBypassHostCheck).
		AssignGroupsToPolicy(FixturePolicyWithGroups.Name, GroupMedia.Name, GroupProductivity.Name).
		WithPolicyBypassHostCheck(FixturePolicyBypassHostCheck.Name).
		WithUser(FixtureUserPairing).
		WithUser(FixtureUserAdmin).
		WithUser(FixtureUserSharedExtra).
		SetUserAccess(FixtureUserSharedExtra.Name, true).
		WithDevice(FixtureDeviceWithOwnerAccess).
		WithDevice(FixtureDeviceWithoutOwnerAccess).
		WithDevice(FixtureDeviceBypassAccess).
		WithDevice(FixtureDevicePairingUsed).
		WithDevice(FixtureDevicePairingExpired).
		WithDevice(FixtureDeviceAdmin).
		WithDevice(FixtureDeviceSuperAdmin).
		WithDevice(FixtureDeviceSharedThird).
		WithDeviceLeaseRule(FixtureLeaseRuleAliceLaptop).
		WithDeviceMaxActiveRule(FixtureMaxActiveRuleAliceLaptop).
		WithAddress(FixtureAddressAlice).
		WithAddress(FixtureAddressAliceDisabled).
		WithAddress(FixtureAddressBob).
		WithAddress(FixtureAddressShared).
		WithAddress(FixtureAddressAdmin).
		WithAddress(FixtureAddressSuperAdmin).
		WithAddress(FixtureAddressSharedThird).
		WithEdgeEntities().
		WithAdversarialEntities().
		WithPairing(FixturePairingBobPending).
		WithPairing(FixturePairingCharlieInvalidated).
		WithPairing(FixturePairingDianaUsed).
		WithPairing(FixturePairingDianaExpired).
		WithPolicyInitialize().
		WithAccessLogEntry(FixtureAccessLogAliceAllow).
		WithAccessLogEntry(FixtureAccessLogBobHostDeny).
		WithAccessLogEntry(FixtureAccessLogUnknownDeny).
		WithAccessLogEntry(FixtureAccessLogSharedIPAllow).
		WithAccessLogEntry(FixtureAccessLogNetworkPolicyAllow).
		WithAccessLogEntry(FixtureAccessLogBypassAllow).
		WithAccessLogEntry(FixtureAccessLogGeoGermanyAPI).
		WithAccessLogEntry(FixtureAccessLogGeoGermanyLogin).
		WithAccessLogEntry(FixtureAccessLogGeoUSA).
		WithAccessLogEntry(FixtureAccessLogGeoSpain)
}
