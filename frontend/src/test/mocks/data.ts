import type { Address, AddressHistoryBucket, AddressHistoryEvent, AddressHistoryResponse, AccessLogCountryStats, DashboardServiceCount, DashboardStats, DashboardTopDeniedIp, DashboardTrafficBucket, Device, DeviceAddressLeaseRule, HostGroupWithMembers, HostSuggestion, HostSuggestionsPage, IgnoredHostSuggestion, KnownHostWithStats, MaxActiveAddressesRule, AccessLogResponse, AccessLogRow, User, UserHostAccessSummary, UserHostDetails, UserHostDetailsGroup, UserHostDetailsHost, GroupRef, PolicyUserAddress, PolicyUserIpSharedUser, PolicyUserIp, PolicyUserEntry, PolicyUserMapAudit, PolicySimulateResult } from '@/lib/api';
import { UserRole } from "@/lib/api";

/**
 * Creates a mock Device object with realistic defaults.
 * @param overrides - Partial Device object to override defaults
 * @returns A Device object
 */
export function createMockDevice(overrides?: Partial<Device>): Device {
  return {
    id: 1,
    name: 'Test Device',
    owner_id: 1,
    api_key_prefix: 'test_',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    device_type: 'static',
    ...overrides,
  };
}

/**
 * Creates a mock User object with realistic defaults.
 * @param overrides - Partial User object to override defaults
 * @returns A User object
 */
export function createMockUser(overrides?: Partial<User>): User {
  return {
    id: 1,
    username: 'testuser',
    display_name: 'Test User',
    email: 'test@example.com',
    role: UserRole.USER,
    must_change_password: false,
    bypass_host_allowlist: false,
    created_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Creates a mock Address object with realistic defaults.
 * @param overrides - Partial Address object to override defaults
 * @returns An Address object
 */
export function createMockAddress(overrides?: Partial<Address>): Address {
  return {
    id: 1,
    device_id: 1,
    ip: '192.168.1.100',
    is_enabled: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Creates a mock DeviceAddressLeaseRule with realistic defaults.
 * @param overrides - Partial DeviceAddressLeaseRule to override defaults
 * @returns A DeviceAddressLeaseRule object
 */
export function createMockDeviceAddressLeaseRule(
  overrides?: Partial<DeviceAddressLeaseRule>,
): DeviceAddressLeaseRule {
  return {
    id: 1,
    device_id: 1,
    enabled: true,
    ttl_seconds: 3600,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Creates a mock MaxActiveAddressesRule with realistic defaults.
 * @param overrides - Partial MaxActiveAddressesRule to override defaults
 * @returns A MaxActiveAddressesRule object
 */
export function createMockMaxActiveAddressesRule(
  overrides?: Partial<MaxActiveAddressesRule>,
): MaxActiveAddressesRule {
  return {
    id: 1,
    device_id: 1,
    enabled: true,
    max_addresses: 3,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
}

/**
 * Creates a mock AccessLogRow with realistic defaults.
 * @param overrides - Partial AccessLogRow to override defaults
 * @returns A AccessLogRow object
 */
export function createMockAccessLogRow(
  overrides?: Partial<AccessLogRow>,
): AccessLogRow {
  return {
    id: 1,
    client_ip: '203.0.113.42',
    outcome: true,
    device_id: 1,
    device_name: 'Test Device',
    created_at: '2024-01-01T12:00:00Z',
    target_host: 'example.com',
    target_uri: '/api/data',
    http_method: 'GET',
    headers: {},
    ...overrides,
  };
}

/**
 * Creates a mock AddressHistoryBucket with realistic defaults.
 */
export function createMockAddressHistoryBucket(
  overrides?: Partial<AddressHistoryBucket>,
): AddressHistoryBucket {
  return {
    timestamp: '2024-01-01T12:00:00Z',
    active_count: 2,
    event_count: 3,
    ...overrides,
  };
}

/**
 * Creates a mock AddressHistoryEvent with realistic defaults.
 */
export function createMockAddressHistoryEvent(
  overrides?: Partial<AddressHistoryEvent>,
): AddressHistoryEvent {
  return {
    id: 1,
    timestamp: '2024-01-01T12:00:00Z',
    ip: '192.168.1.100',
    is_enabled: true,
    source: 'heartbeat',
    device_id: 1,
    device_name: 'Test Device',
    ...overrides,
  };
}

/**
 * Creates a mock AddressHistoryResponse with realistic defaults.
 */
export function createMockAddressHistoryResponse(
  overrides?: Partial<AddressHistoryResponse>,
): AddressHistoryResponse {
  return {
    buckets: [
      createMockAddressHistoryBucket({ timestamp: '2024-01-01T10:00:00Z', active_count: 1, event_count: 1 }),
      createMockAddressHistoryBucket({ timestamp: '2024-01-01T11:00:00Z', active_count: 2, event_count: 2 }),
      createMockAddressHistoryBucket({ timestamp: '2024-01-01T12:00:00Z', active_count: 3, event_count: 1 }),
    ],
    events: [
      createMockAddressHistoryEvent({ id: 3, timestamp: '2024-01-01T10:30:00Z', ip: '10.0.0.1', is_enabled: true, source: 'heartbeat' }),
      createMockAddressHistoryEvent({ id: 2, timestamp: '2024-01-01T11:00:00Z', ip: '10.0.0.2', is_enabled: true, source: 'manual' }),
      createMockAddressHistoryEvent({ id: 1, timestamp: '2024-01-01T11:45:00Z', ip: '10.0.0.1', is_enabled: false, source: 'expiry' }),
    ],
    total_events: 3,
    next_cursor: null,
    ...overrides,
  };
}

/**
 * Creates a mock AccessLogResponse with realistic defaults.
 * @param overrides - Partial AccessLogResponse to override defaults
 * @returns A AccessLogResponse object
 */
export function createMockAccessLogResponse(
  overrides?: Partial<AccessLogResponse>,
): AccessLogResponse {
  return {
    total: 1,
    next_cursor: null,
    rows: [createMockAccessLogRow()],
    ...overrides,
  };
}

// ─── Dashboard mock data ─────────────────────────────────────────────────────

export function createMockDashboardStats(
  overrides?: Partial<DashboardStats>,
): DashboardStats {
  return {
    total_requests: 150,
    allowed_count: 120,
    denied_count: 30,
    unique_ips: 8,
    avg_duration_us: 1250,
    ...overrides,
  };
}

export function createMockDashboardTrafficBucket(
  overrides?: Partial<DashboardTrafficBucket>,
): DashboardTrafficBucket {
  return {
    timestamp: '2024-01-01T12:00:00Z',
    allow_count: 40,
    deny_count: 10,
    ...overrides,
  };
}

export function createMockDashboardServiceCount(
  overrides?: Partial<DashboardServiceCount>,
): DashboardServiceCount {
  return {
    host: 'app.example.com',
    allow_count: 80,
    deny_count: 15,
    ...overrides,
  };
}

export function createMockDashboardTopDeniedIp(
  overrides?: Partial<DashboardTopDeniedIp>,
): DashboardTopDeniedIp {
  return {
    ip: '203.0.113.42',
    count: 25,
    ...overrides,
  };
}

export function createMockAccessLogCountryStats(
  overrides?: Partial<AccessLogCountryStats>,
): AccessLogCountryStats {
  return {
    country_code: 'US',
    country_name: 'United States',
    continent_code: 'NA',
    total: 100,
    allowed: 80,
    denied: 20,
    ...overrides,
  };
}

// ─── Host-access mock data ───────────────────────────────────────────────────

export function createMockGroupRef(overrides?: Partial<GroupRef>): GroupRef {
  return {
    id: 1,
    name: 'default-group',
    ...overrides,
  };
}

export function createMockUserHostAccessSummary(
  overrides?: Partial<UserHostAccessSummary>,
): UserHostAccessSummary {
  return {
    id: 1,
    display_name: 'Test User',
    email: 'test@example.com',
    role: UserRole.USER,
    bypass: false,
    direct_host_count: 0,
    groups: [],
    ...overrides,
  };
}

export function createMockUserHostDetailsHost(
  overrides?: Partial<UserHostDetailsHost>,
): UserHostDetailsHost {
  return {
    id: 1,
    fqdn: 'app.example.com',
    directly_granted: false,
    icon: null,
    via_group: null,
    ...overrides,
  };
}

export function createMockUserHostDetailsGroup(
  overrides?: Partial<UserHostDetailsGroup>,
): UserHostDetailsGroup {
  return {
    id: 1,
    name: 'default-group',
    granted: false,
    icon: null,
    hosts: [],
    ...overrides,
  };
}

export function createMockUserHostDetails(
  overrides?: Partial<UserHostDetails>,
): UserHostDetails {
  return {
    id: 1,
    display_name: 'Test User',
    email: 'test@example.com',
    role: UserRole.USER,
    bypass: false,
    groups: [],
    hosts: [],
    ...overrides,
  };
}

export function createMockKnownHostWithStats(
  overrides?: Partial<KnownHostWithStats>,
): KnownHostWithStats {
  return {
    id: 1,
    fqdn: 'host.lan',
    icon: null,
    created_at: '2026-01-01T00:00:00Z',
    user_count: 0,
    groups: [],
    ...overrides,
  };
}

export function createMockHostGroupWithMembers(
  overrides?: Partial<HostGroupWithMembers>,
): HostGroupWithMembers {
  return {
    id: 1,
    name: 'Test Group',
    description: null,
    icon: null,
    color: null,
    created_at: '2026-01-01T00:00:00Z',
    hosts: [],
    member_ids: [],
    ...overrides,
  };
}

export function createMockHostSuggestion(
  overrides?: Partial<HostSuggestion>,
): HostSuggestion {
  return {
    fqdn: 'unknown.lan',
    first_seen: '2026-04-01T00:00:00Z',
    allowed_hits: 5,
    denied_hits: 0,
    ...overrides,
  };
}

export function createMockIgnoredHostSuggestion(
  overrides?: Partial<IgnoredHostSuggestion>,
): IgnoredHostSuggestion {
  return {
    id: 1,
    fqdn: 'ignored.lan',
    created_at: '2026-04-01T00:00:00Z',
    ...overrides,
  };
}

export function createMockHostSuggestionsPage(
  overrides?: Partial<HostSuggestionsPage>,
): HostSuggestionsPage {
  return {
    suggestions: [],
    ignored: [],
    ...overrides,
  };
}

// ─── Policy-audit mock data ──────────────────────────────────────────────────

export function createMockPolicyUserAddress(
  overrides?: Partial<PolicyUserAddress>,
): PolicyUserAddress {
  return {
    address_id: 1,
    device_id: 1,
    device_name: 'Test Device',
    updated_at: '2026-01-01T10:00:00Z',
    ...overrides,
  };
}

export function createMockPolicyUserIpSharedUser(
  overrides?: Partial<PolicyUserIpSharedUser>,
): PolicyUserIpSharedUser {
  return {
    user_id: 2,
    username: 'other',
    user_name: 'Other User',
    devices: [],
    ...overrides,
  };
}

export function createMockPolicyUserIp(
  overrides?: Partial<PolicyUserIp>,
): PolicyUserIp {
  return {
    ip: '192.168.1.10',
    shared_with_users: [],
    bypass_at_ip: false,
    effective_hosts: ['app.home.lan', 'media.home.lan'],
    trimmed_hosts: [],
    addresses: [createMockPolicyUserAddress()],
    ...overrides,
  };
}

export function createMockPolicyUserEntry(
  overrides?: Partial<PolicyUserEntry>,
): PolicyUserEntry {
  return {
    user_id: 1,
    user_name: 'alice',
    is_admin: false,
    bypass_allowlist: false,
    on_shared_ip: false,
    intersection_applied: false,
    device_count: 1,
    ip_count: 1,
    allowed_host_count: 2,
    last_seen_at: '2026-01-01T10:00:00Z',
    user_allowed_hosts: ['app.home.lan', 'media.home.lan'],
    ips: [createMockPolicyUserIp()],
    ...overrides,
  };
}

// Default audit contains one user of each status so all StatusBadge branches
// are covered by the happy-path handler without per-test overrides.
export function createMockPolicyUserMapAudit(
  overrides?: Partial<PolicyUserMapAudit>,
): PolicyUserMapAudit {
  return {
    refreshed_at: '2026-01-01T09:00:00Z',
    refresh_duration_ms: 42,
    total_ip_count: 3,
    total_device_count: 3,
    total_host_count: 5,
    shared_ip_count: 1,
    total_network_policy_count: 0,
    network_policies: [],
    users: [
      createMockPolicyUserEntry({ user_id: 1, user_name: 'alice', bypass_allowlist: false }),
      createMockPolicyUserEntry({
        user_id: 2,
        user_name: 'bob',
        bypass_allowlist: true,
        ips: [],
        ip_count: 0,
        device_count: 0,
        allowed_host_count: 0,
        user_allowed_hosts: [],
      }),
      createMockPolicyUserEntry({
        user_id: 3,
        user_name: 'carol',
        bypass_allowlist: false,
        ips: [],
        ip_count: 0,
        device_count: 0,
        allowed_host_count: 0,
        user_allowed_hosts: [],
      }),
    ],
    ...overrides,
  };
}

export function createMockPolicySimulateResult(
  overrides?: Partial<PolicySimulateResult>,
): PolicySimulateResult {
  return {
    ip: '192.168.1.10',
    host: 'app.home.lan',
    allowed: true,
    deny_reason: null,
    ...overrides,
  };
}
