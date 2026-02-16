import type { Address, Device, User } from '@/lib/api';

/**
 * Creates a mock Device object with realistic defaults.
 * @param overrides - Partial Device object to override defaults
 * @returns A Device object
 */
export function createMockDevice(overrides?: Partial<Device>): Device {
  return {
    id: 1,
    name: 'Test Device',
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
    status: true,
    created_at: new Date('2024-01-01T00:00:00Z'),
    updated_at: new Date('2024-01-01T00:00:00Z'),
    ...overrides,
  };
}
