import type { Address, AddressHistoryBucket, AddressHistoryEvent, AddressHistoryResponse, Device, DeviceAddressLeaseRule, RequestAuditLogResponse, RequestAuditLogRow, User } from '@/lib/api';
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
 * Creates a mock RequestAuditLogRow with realistic defaults.
 * @param overrides - Partial RequestAuditLogRow to override defaults
 * @returns A RequestAuditLogRow object
 */
export function createMockRequestAuditLogRow(
  overrides?: Partial<RequestAuditLogRow>,
): RequestAuditLogRow {
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
 * Creates a mock RequestAuditLogResponse with realistic defaults.
 * @param overrides - Partial RequestAuditLogResponse to override defaults
 * @returns A RequestAuditLogResponse object
 */
export function createMockRequestAuditLogResponse(
  overrides?: Partial<RequestAuditLogResponse>,
): RequestAuditLogResponse {
  return {
    total: 1,
    next_cursor: null,
    rows: [createMockRequestAuditLogRow()],
    ...overrides,
  };
}
