import { http, HttpResponse, type JsonBodyType } from 'msw';
import type { Address, AddressHistoryResponse, AccessLogCountryStats, CreateDeviceResponse, DashboardServiceCount, DashboardStats, DashboardTopDeniedIp, DashboardTrafficBucket, Device, DeviceAddressLeaseRule, DeviceOwnerGroup, DevicePairing, GroupDetailWithUsers, Host, HostSuggestionsPage, IgnoredHostSuggestion, MaxActiveAddressesRule, AccessLogResponse, NetworkPolicyListItem, NetworkPolicyDetail, User, UserListItem, UserAccessDetail, PolicyUserMapAudit, PolicySimulateResult } from '@/lib/api';
import { createMockAddress, createMockAddressHistoryResponse, createMockAccessLogCountryStats, createMockDashboardServiceCount, createMockDashboardStats, createMockDashboardTopDeniedIp, createMockDashboardTrafficBucket, createMockDevice, createMockDeviceAddressLeaseRule, createMockDeviceOwnerGroup, createMockDevicePairing, createMockHostSuggestionsPage, createMockIgnoredHostSuggestion, createMockMaxActiveAddressesRule, createMockAccessLogResponse, createMockNetworkPolicyListItem, createMockNetworkPolicyDetail, createMockUser, createMockUserListItem, createMockUserAccessDetail, createMockPolicyUserMapAudit, createMockPolicySimulateResult } from './data';

const BASE = '/api/v1';

// ─── Endpoint path constants ──────────────────────────────────────────────────
// All endpoint strings live here. Never hardcode them in test files.
export const endpoints = {
    usersHostAccess: `${BASE}/admin/access/users`,
    userHostDetails: `${BASE}/admin/access/users/:userId`,
    setUserHostGrants: `${BASE}/admin/access/users/:userId/grants`,
    devices: `${BASE}/devices`,
    deviceById: `${BASE}/devices/:deviceId`,
    deviceAddresses: `${BASE}/devices/:deviceId/addresses`,
    deviceAddressById: `${BASE}/devices/:deviceId/addresses/:addressId`,
    deviceHeartbeat: `${BASE}/devices/:deviceId/heartbeat`,
    addressHistory: `${BASE}/address-history`,
    deviceAddressLeaseRule: `${BASE}/devices/:deviceId/rules/address-lease`,
    maxActiveAddressesRule: `${BASE}/devices/:deviceId/rules/max-active-addresses`,
    regenerateApiKey: `${BASE}/devices/:deviceId/api-key/regenerate`,
    deleteApiKey: `${BASE}/devices/:deviceId/api-key`,
    authMe: `${BASE}/auth/me`,
    authLogin: `${BASE}/auth/login`,
    adminUsers: `${BASE}/admin/users`,
    adminUserById: `${BASE}/admin/users/:userId`,
    promoteUser: `${BASE}/admin/users/:userId/promote`,
    demoteUser: `${BASE}/admin/users/:userId/demote`,
    updateMe: `${BASE}/users/me`,
    changePassword: `${BASE}/users/me/password`,
    accessLog: `${BASE}/access-log`,
    accessLogDenyReasons: `${BASE}/access-log/deny-reasons`,
    accessLogByCountry: `${BASE}/access-log/stats/by-country`,
    dashboardStats: `${BASE}/dashboard/stats`,
    dashboardTraffic: `${BASE}/dashboard/traffic`,
    dashboardServices: `${BASE}/dashboard/services`,
    dashboardTopDeniedIps: `${BASE}/dashboard/top-denied-ips`,
    adminHosts: `${BASE}/admin/access/hosts`,
    adminHostsReconcile: `${BASE}/admin/access/hosts/reconcile`,
    adminHostGroups: `${BASE}/admin/access/host-groups`,
    adminHostGroupsReconcile: `${BASE}/admin/access/host-groups/reconcile`,
    adminHostSuggestions: `${BASE}/admin/access/host-suggestions`,
    adminHostSuggestionsIgnore: `${BASE}/admin/access/host-suggestions/ignore`,
    adminHostSuggestionsIgnoreByFqdn: `${BASE}/admin/access/host-suggestions/ignore/:fqdn`,
    policyMap:      `${BASE}/admin/policy-map`,
    policySimulate: `${BASE}/admin/policy-simulate`,
    networkPolicies: `${BASE}/admin/access/network-policies`,
    networkPolicyById: `${BASE}/admin/access/network-policies/:id`,
    networkPolicyGrants: `${BASE}/admin/access/network-policies/:id/grants`,
    devicePairings: `${BASE}/devices/:id/pairings`,
    devicePairingById: `${BASE}/devices/:id/pairings/:pairingId`,
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
    list: (groups?: DeviceOwnerGroup[]) =>
        http.get(endpoints.devices, () =>
            HttpResponse.json(groups ?? [createMockDeviceOwnerGroup()])),

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

    deleteApiKey: {
        success: () =>
            http.delete(endpoints.deleteApiKey, () => responses.noContent()),
        notFound: () =>
            http.delete(endpoints.deleteApiKey, () => responses.notFound()),
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
    maxActiveAddresses: {
        get: {
            success: (override?: Partial<MaxActiveAddressesRule>) =>
                http.get(endpoints.maxActiveAddressesRule, () =>
                    HttpResponse.json({ ...createMockMaxActiveAddressesRule(), ...override })),
            notFound: () =>
                http.get(endpoints.maxActiveAddressesRule, () =>
                    responses.notFound({ error: 'Not Found' })),
        },
        put: {
            success: (override?: Partial<MaxActiveAddressesRule>) =>
                http.put(endpoints.maxActiveAddressesRule, ({ params }) =>
                    HttpResponse.json({
                        ...createMockMaxActiveAddressesRule({ device_id: Number(params.deviceId) }),
                        ...override,
                    })),
        },
        delete: {
            success: () =>
                http.delete(endpoints.maxActiveAddressesRule, () => responses.noContent()),
        },
    },
};

// ─── Access log handlers ───────────────────────────────────────────────
export const accessLogHandlers = {
    list: (override?: AccessLogResponse) =>
        http.get(endpoints.accessLog, () =>
            HttpResponse.json(override ?? createMockAccessLogResponse())),

    denyReasons: (reasons?: string[]) =>
        http.get(endpoints.accessLogDenyReasons, () =>
            HttpResponse.json(reasons ?? ['invalid_token', 'ip_not_registered', 'no_device_match'])),

    byCountry: (stats?: AccessLogCountryStats[]) =>
        http.get(endpoints.accessLogByCountry, () =>
            HttpResponse.json(
                stats ?? [
                    createMockAccessLogCountryStats({ country_code: 'US', country_name: 'United States', total: 100, allowed: 80, denied: 20 }),
                    createMockAccessLogCountryStats({ country_code: 'DE', country_name: 'Germany', total: 50, allowed: 45, denied: 5 }),
                    createMockAccessLogCountryStats({ country_code: 'CN', country_name: 'China', total: 75, allowed: 5, denied: 70 }),
                ],
            )),
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

// ─── Host-access handlers ──────────────────────────────────────────────────────
export const hostAccessHandlers = {
    listUsersHostAccess: {
        success: (summaries?: UserListItem[]) =>
            http.get(endpoints.usersHostAccess, () =>
                HttpResponse.json(summaries ?? [createMockUserListItem()])),
        serverError: () =>
            http.get(endpoints.usersHostAccess, () => responses.serverError()),
    },
    userHostDetails: {
        success: (details?: UserAccessDetail) =>
            http.get(endpoints.userHostDetails, () =>
                HttpResponse.json(details ?? createMockUserAccessDetail())),
    },
    setUserHostGrants: {
        success: () =>
            http.put(endpoints.setUserHostGrants, () => responses.noContent()),
    },
    listKnownHosts: {
        success: (hosts: Host[] = []) =>
            http.get(endpoints.adminHosts, () => HttpResponse.json({ hosts })),
        serverError: () =>
            http.get(endpoints.adminHosts, () => responses.serverError()),
    },
    listHostGroups: {
        success: (groups: GroupDetailWithUsers[] = []) =>
            http.get(endpoints.adminHostGroups, () => HttpResponse.json({ groups })),
        serverError: () =>
            http.get(endpoints.adminHostGroups, () => responses.serverError()),
    },
    listHostSuggestions: {
        success: (page?: HostSuggestionsPage) =>
            http.get(endpoints.adminHostSuggestions, () =>
                HttpResponse.json(page ?? createMockHostSuggestionsPage())),
        serverError: () =>
            http.get(endpoints.adminHostSuggestions, () => responses.serverError()),
    },
    reconcileKnownHosts: {
        success: () =>
            http.post(endpoints.adminHostsReconcile, () => responses.noContent()),
    },
    reconcileHostGroups: {
        success: () =>
            http.post(endpoints.adminHostGroupsReconcile, () => responses.noContent()),
    },
    ignoreSuggestion: {
        success: (entry?: IgnoredHostSuggestion) =>
            http.post(endpoints.adminHostSuggestionsIgnore, () =>
                responses.created(entry ?? createMockIgnoredHostSuggestion())),
    },
    unignoreSuggestion: {
        success: () =>
            http.delete(endpoints.adminHostSuggestionsIgnoreByFqdn, () => responses.noContent()),
    },
};

// ─── Policy-audit handlers ────────────────────────────────────────────────────
export const policyAuditHandlers = {
    policyMap: {
        success: (override?: Partial<PolicyUserMapAudit>) =>
            http.get(endpoints.policyMap, () =>
                HttpResponse.json(createMockPolicyUserMapAudit(override))),
        serverError: () =>
            http.get(endpoints.policyMap, () => responses.serverError()),
    },
    simulate: {
        allowed: (override?: Partial<PolicySimulateResult>) =>
            http.get(endpoints.policySimulate, () =>
                HttpResponse.json(createMockPolicySimulateResult({ allowed: true, ...override }))),
        denied: (denyReason = 'ip_not_registered') =>
            http.get(endpoints.policySimulate, () =>
                HttpResponse.json(createMockPolicySimulateResult({
                    allowed: false,
                    deny_reason: denyReason as PolicySimulateResult['deny_reason'],
                }))),
    },
};

// ─── Network policy handlers ──────────────────────────────────────────────────
export const networkPolicyHandlers = {
    list: {
        success: (policies?: NetworkPolicyListItem[]) =>
            http.get(endpoints.networkPolicies, () =>
                HttpResponse.json(policies ?? [createMockNetworkPolicyListItem()])),
        serverError: () =>
            http.get(endpoints.networkPolicies, () => responses.serverError()),
    },
    get: {
        success: (override?: Partial<NetworkPolicyDetail>) =>
            http.get(endpoints.networkPolicyById, () =>
                HttpResponse.json(createMockNetworkPolicyDetail(override))),
        notFound: () =>
            http.get(endpoints.networkPolicyById, () => responses.notFound()),
    },
    create: {
        success: (override?: Partial<NetworkPolicyDetail>) =>
            http.post(endpoints.networkPolicies, async ({ request }) => {
                const body = (await request.json()) as Partial<NetworkPolicyDetail>;
                return responses.created(createMockNetworkPolicyDetail({ ...body, ...override }));
            }),
    },
    update: {
        success: () =>
            http.put(endpoints.networkPolicyById, () => responses.noContent()),
    },
    delete: {
        success: () =>
            http.delete(endpoints.networkPolicyById, () => responses.noContent()),
    },
    updateGrants: {
        success: () =>
            http.put(endpoints.networkPolicyGrants, () => responses.noContent()),
    },
};

// ─── Device pairing handlers ──────────────────────────────────────────────────
export const devicePairingHandlers = {
    list: {
        success: (pairings?: DevicePairing[]) =>
            http.get(endpoints.devicePairings, () =>
                HttpResponse.json(pairings ?? [createMockDevicePairing()])),
        empty: () =>
            http.get(endpoints.devicePairings, () => HttpResponse.json([])),
    },
    create: {
        success: (override?: Partial<DevicePairing>) =>
            http.post(endpoints.devicePairings, () =>
                responses.created(createMockDevicePairing(override))),
    },
    delete: {
        success: () =>
            http.delete(endpoints.devicePairingById, () => responses.noContent()),
    },
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
    ruleHandlers.maxActiveAddresses.get.success(),
    ruleHandlers.maxActiveAddresses.put.success(),
    ruleHandlers.maxActiveAddresses.delete.success(),
    // Access log
    accessLogHandlers.list(),
    accessLogHandlers.denyReasons(),
    accessLogHandlers.byCountry(),
    // Dashboard
    dashboardHandlers.stats(),
    dashboardHandlers.traffic(),
    dashboardHandlers.services(),
    dashboardHandlers.topDeniedIps(),
    // Host access
    hostAccessHandlers.listUsersHostAccess.success(),
    hostAccessHandlers.userHostDetails.success(),
    hostAccessHandlers.setUserHostGrants.success(),
    hostAccessHandlers.listKnownHosts.success(),
    hostAccessHandlers.listHostGroups.success(),
    hostAccessHandlers.listHostSuggestions.success(),
    hostAccessHandlers.reconcileKnownHosts.success(),
    hostAccessHandlers.reconcileHostGroups.success(),
    // Network policies
    networkPolicyHandlers.list.success(),
    networkPolicyHandlers.get.success(),
    networkPolicyHandlers.update.success(),
    networkPolicyHandlers.delete.success(),
    networkPolicyHandlers.updateGrants.success(),
    // Device pairing
    devicePairingHandlers.list.success(),
    devicePairingHandlers.create.success(),
    devicePairingHandlers.delete.success(),
    // Policy audit
    policyAuditHandlers.policyMap.success(),
    policyAuditHandlers.simulate.allowed(),
];
