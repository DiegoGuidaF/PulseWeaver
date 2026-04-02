import type { Address, AddressHistoryBucket, AddressHistoryEvent, AddressHistoryResponse, AccessLogCountryStats, DashboardServiceCount, DashboardStats, DashboardTopDeniedIp, DashboardTrafficBucket, Device, DeviceAddressLeaseRule, MaxActiveAddressesRule, AccessLogResponse, AccessLogRow, User } from '@/lib/api';
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
    api_key_prefix: 'test_',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    device_type: 'generic',
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
