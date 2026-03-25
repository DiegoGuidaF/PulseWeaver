import { http, HttpResponse, type JsonBodyType } from 'msw';
import type { Address, AddressHistoryResponse, CreateDeviceResponse, DashboardServiceCount, DashboardStats, DashboardTopDeniedIp, DashboardTrafficBucket, Device, DeviceAddressLeaseRule, RequestAuditLogResponse, User } from '@/lib/api';
import { createMockAddress, createMockAddressHistoryResponse, createMockDashboardServiceCount, createMockDashboardStats, createMockDashboardTopDeniedIp, createMockDashboardTrafficBucket, createMockDevice, createMockDeviceAddressLeaseRule, createMockRequestAuditLogResponse, createMockUser } from './data';

const BASE = '/api/v1';

// ─── Endpoint path constants ──────────────────────────────────────────────────
// All endpoint strings live here. Never hardcode them in test files.
export const endpoints = {
    devices: `${BASE}/devices`,
    deviceById: `${BASE}/devices/:deviceId`,
    deviceAddresses: `${BASE}/devices/:deviceId/addresses`,
    deviceAddressById: `${BASE}/devices/:deviceId/addresses/:addressId`,
    deviceHeartbeat: `${BASE}/devices/:deviceId/heartbeat`,
    addressHistory: `${BASE}/address-history`,
    deviceAddressLeaseRule: `${BASE}/devices/:deviceId/rules/address_lease`,
    regenerateApiKey: `${BASE}/devices/:deviceId/api-key/regenerate`,
    authMe: `${BASE}/auth/me`,
    authLogin: `${BASE}/auth/login`,
    adminUsers: `${BASE}/admin/users`,
    adminUserById: `${BASE}/admin/users/:userId`,
    promoteUser: `${BASE}/admin/users/:userId/promote`,
    demoteUser: `${BASE}/admin/users/:userId/demote`,
    updateMe: `${BASE}/users/me`,
    changePassword: `${BASE}/users/me/password`,
    requestAuditLog: `${BASE}/request-audit-log`,
    requestAuditLogDenyReasons: `${BASE}/request-audit-log/deny-reasons`,
    dashboardStats: `${BASE}/dashboard/stats`,
    dashboardTraffic: `${BASE}/dashboard/traffic`,
    dashboardServices: `${BASE}/dashboard/services`,
    dashboardTopDeniedIps: `${BASE}/dashboard/top-denied-ips`,
} as const;

// ─── Response helpers ─────────────────────────────────────────────────────────
export const responses = {
    ok: <T extends JsonBodyType>(data: T) =>
        HttpResponse.json(data),

    created: <T extends JsonBodyType>(data: T) =>
        HttpResponse.json(data, { status: 201 }),

    noContent: () =>
        new HttpResponse(null, { status: 204 }),

    badRequest: (data: JsonBodyType = { error: 'Bad Request' }) =>
        HttpResponse.json(data, { status: 400 }),

    unauthorized: (data: JsonBodyType = { message: 'Unauthorized' }) =>
        HttpResponse.json(data, { status: 401 }),

    forbidden: (data: JsonBodyType = { message: 'Forbidden' }) =>
        HttpResponse.json(data, { status: 403 }),

    notFound: (data: JsonBodyType = { message: 'Not Found' }) =>
        HttpResponse.json(data, { status: 404 }),

    serverError: (data: JsonBodyType = { message: 'Internal Server Error' }) =>
        HttpResponse.json(data, { status: 500 }),

    custom: (data: JsonBodyType, status: number) =>
        HttpResponse.json(data, { status }),
};

// ─── Auth handlers ────────────────────────────────────────────────────────────
export const authHandlers = {
    me: {
        success: (override?: Partial<User>) =>
            http.get(endpoints.authMe, () =>
                HttpResponse.json({ ...createMockUser(), ...override })),
        unauthenticated: () =>
            http.get(endpoints.authMe, () => responses.unauthorized()),
    },
    login: {
        success: (override?: Partial<User>) =>
            http.post(endpoints.authLogin, () =>
                HttpResponse.json({ ...createMockUser(), ...override })),
        invalidCredentials: () =>
            http.post(endpoints.authLogin, () =>
                responses.unauthorized({ message: 'Invalid credentials' })),
    },
    listUsers: {
        success: (users?: User[]) =>
            http.get(endpoints.adminUsers, () =>
                HttpResponse.json(users ?? [createMockUser()])),
    },
    updateMe: {
        success: (override?: Partial<User>) =>
            http.patch(endpoints.updateMe, () =>
                HttpResponse.json({ ...createMockUser(), ...override })),
    },
    changePassword: {
        success: () =>
            http.post(endpoints.changePassword, () => responses.noContent()),
    },
    promoteUser: {
        success: (override?: Partial<User>) =>
            http.post(endpoints.promoteUser, () =>
                HttpResponse.json({ ...createMockUser({ role: 'admin' }), ...override })),
    },
    demoteUser: {
        success: (override?: Partial<User>) =>
            http.post(endpoints.demoteUser, () =>
                HttpResponse.json({ ...createMockUser({ role: 'user' }), ...override })),
    },
    deleteUser: {
        success: () =>
            http.delete(endpoints.adminUserById, () => responses.noContent()),
    },
};

// ─── Device handlers ──────────────────────────────────────────────────────────
export const deviceHandlers = {
    list: (devices?: Device[]) =>
        http.get(endpoints.devices, () =>
            HttpResponse.json(devices ?? [createMockDevice()])),

    getById: (override?: Partial<Device>) =>
        http.get(endpoints.deviceById, ({ params }) =>
            HttpResponse.json(createMockDevice({ id: Number(params.deviceId), ...override }))),

    create: {
        success: (override?: Partial<CreateDeviceResponse>) =>
            http.post(endpoints.devices, () =>
                HttpResponse.json(
                    {
                        device: createMockDevice(),
                        api_key: 'test_api_key_12345678901234567890123456789012',
                        ...override,
                    },
                    { status: 201 }
                )),
        conflict: () =>
            http.post(endpoints.devices, () =>
                responses.custom({ error: 'Device name already in use' }, 409)),
    },

    delete: {
        success: () =>
            http.delete(endpoints.deviceById, () => responses.noContent()),
        notFound: () =>
            http.delete(endpoints.deviceById, () => responses.notFound()),
    },

    regenerateApiKey: {
        success: (override?: { api_key?: string }) =>
            http.post(endpoints.regenerateApiKey, ({ params }) =>
                HttpResponse.json({
                    device: createMockDevice({ id: Number(params.deviceId) }),
                    api_key: 'regenerated_key_abc123xyz789',
                    ...override,
                })),
    },
};

// ─── Address handlers ─────────────────────────────────────────────────────────
export const addressHandlers = {
    list: (addresses?: Address[]) =>
        http.get(endpoints.deviceAddresses, () =>
            HttpResponse.json(addresses ?? [createMockAddress()])),

    create: {
        success: (override?: Partial<Address>) =>
            http.post(endpoints.deviceAddresses, () =>
                HttpResponse.json({ ...createMockAddress(), ...override }, { status: 201 })),
    },

    heartbeat: {
        success: (override?: Partial<Address>) =>
            http.post(endpoints.deviceHeartbeat, () =>
                HttpResponse.json({
                    ...createMockAddress({ ip: '192.168.1.100', is_enabled: true }),
                    ...override,
                })),
    },

    disable: {
        success: (override?: Partial<Address>) =>
            http.delete(endpoints.deviceAddressById, () =>
                HttpResponse.json({ ...createMockAddress({ is_enabled: false }), ...override })),
    },

    history: {
        success: (override?: AddressHistoryResponse) =>
            http.get(endpoints.addressHistory, () =>
                HttpResponse.json(override ?? createMockAddressHistoryResponse())),
        empty: () =>
            http.get(endpoints.addressHistory, () =>
                HttpResponse.json({ buckets: [], events: [], total_events: 0, next_cursor: null })),
    },
};

// ─── Rule handlers ────────────────────────────────────────────────────────────
export const ruleHandlers = {
    addressLease: {
        get: {
            success: (override?: Partial<DeviceAddressLeaseRule>) =>
                http.get(endpoints.deviceAddressLeaseRule, () =>
                    HttpResponse.json({ ...createMockDeviceAddressLeaseRule(), ...override })),
            notFound: () =>
                http.get(endpoints.deviceAddressLeaseRule, () =>
                    responses.notFound({ error: 'Not Found' })),
        },
        put: {
            success: (override?: Partial<DeviceAddressLeaseRule>) =>
                http.put(endpoints.deviceAddressLeaseRule, ({ params }) =>
                    HttpResponse.json({
                        ...createMockDeviceAddressLeaseRule({ device_id: Number(params.deviceId) }),
                        ...override,
                    })),
        },
        delete: {
            success: () =>
                http.delete(endpoints.deviceAddressLeaseRule, () => responses.noContent()),
        },
    },
};

// ─── Request audit log handlers ───────────────────────────────────────────────
export const requestAuditLogHandlers = {
    list: (override?: RequestAuditLogResponse) =>
        http.get(endpoints.requestAuditLog, () =>
            HttpResponse.json(override ?? createMockRequestAuditLogResponse())),

    denyReasons: (reasons?: string[]) =>
        http.get(endpoints.requestAuditLogDenyReasons, () =>
            HttpResponse.json(reasons ?? ['invalid_token', 'ip_not_registered', 'no_device_match'])),
};

// ─── Dashboard handlers ──────────────────────────────────────────────────────
export const dashboardHandlers = {
    stats: (override?: DashboardStats) =>
        http.get(endpoints.dashboardStats, () =>
            HttpResponse.json(override ?? createMockDashboardStats())),

    traffic: (buckets?: DashboardTrafficBucket[]) =>
        http.get(endpoints.dashboardTraffic, () =>
            HttpResponse.json({
                buckets: buckets ?? [
                    createMockDashboardTrafficBucket({ timestamp: '2024-01-01T10:00:00Z', allow_count: 30, deny_count: 5 }),
                    createMockDashboardTrafficBucket({ timestamp: '2024-01-01T11:00:00Z', allow_count: 45, deny_count: 12 }),
                    createMockDashboardTrafficBucket({ timestamp: '2024-01-01T12:00:00Z', allow_count: 40, deny_count: 10 }),
                ],
            })),

    services: (services?: DashboardServiceCount[]) =>
        http.get(endpoints.dashboardServices, () =>
            HttpResponse.json({
                services: services ?? [
                    createMockDashboardServiceCount({ host: 'app.example.com', allow_count: 80, deny_count: 15 }),
                    createMockDashboardServiceCount({ host: 'api.example.com', allow_count: 40, deny_count: 5 }),
                ],
            })),

    topDeniedIps: (ips?: DashboardTopDeniedIp[]) =>
        http.get(endpoints.dashboardTopDeniedIps, () =>
            HttpResponse.json({
                ips: ips ?? [
                    createMockDashboardTopDeniedIp({ ip: '203.0.113.42', count: 25 }),
                    createMockDashboardTopDeniedIp({ ip: '198.51.100.7', count: 12 }),
                ],
            })),
};

// ─── Default happy-path handlers (registered globally in setup.ts) ────────────
// Every test starts in a fully-loaded state. Only call server.use() for deviations.
export const defaultHandlers = [
    // Auth
    authHandlers.me.success(),
    authHandlers.login.success(),
    authHandlers.listUsers.success(),
    authHandlers.updateMe.success(),
    authHandlers.changePassword.success(),
    authHandlers.promoteUser.success(),
    authHandlers.demoteUser.success(),
    authHandlers.deleteUser.success(),
    // Devices
    deviceHandlers.list(),
    deviceHandlers.getById(),
    deviceHandlers.create.success(),
    deviceHandlers.delete.success(),
    deviceHandlers.regenerateApiKey.success(),
    // Addresses
    addressHandlers.list(),
    addressHandlers.create.success(),
    addressHandlers.heartbeat.success(),
    addressHandlers.disable.success(),
    addressHandlers.history.success(),
    // Rules
    ruleHandlers.addressLease.get.success(),
    ruleHandlers.addressLease.put.success(),
    ruleHandlers.addressLease.delete.success(),
    // Request audit log
    requestAuditLogHandlers.list(),
    requestAuditLogHandlers.denyReasons(),
    // Dashboard
    dashboardHandlers.stats(),
    dashboardHandlers.traffic(),
    dashboardHandlers.services(),
    dashboardHandlers.topDeniedIps(),
];
