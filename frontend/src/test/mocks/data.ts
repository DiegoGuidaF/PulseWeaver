import type { Address, Device, DeviceAddressLeaseRule, User } from '@/lib/api';
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
    created_at: new Date('2024-01-01T00:00:00Z'),
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
    created_at: new Date('2024-01-01T00:00:00Z'),
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
    created_at: new Date('2024-01-01T00:00:00Z'),
    updated_at: new Date('2024-01-01T00:00:00Z'),
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
    created_at: new Date('2024-01-01T00:00:00Z'),
    updated_at: new Date('2024-01-01T00:00:00Z'),
    ...overrides,
  };
}
