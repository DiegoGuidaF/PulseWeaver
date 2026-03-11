import {
    type DefaultBodyType,
    http,
    HttpResponse,
    type HttpResponseResolver,
    type JsonBodyType,
    type PathParams
} from "msw";
import {createMockAddress, createMockDevice, createMockDeviceAddressLeaseRule, createMockUser} from "@/test/mocks/data.ts";

const BASE_URL = '/api/v1';

const apiEndpoints = {
    devices: `${BASE_URL}/devices`,
    deviceById: `${BASE_URL}/devices/:deviceId`,
    deviceAddresses: `${BASE_URL}/devices/:deviceId/addresses`,
    deleteDeviceAddresses: `${BASE_URL}/devices/:deviceId/addresses/:addressId`,
    deviceHeartbeat: `${BASE_URL}/devices/:deviceId/heartbeat`,
    deviceAddressLeaseRule: `${BASE_URL}/devices/:deviceId/rules/address_lease`,
    authMe: `${BASE_URL}/auth/me`,
    authLogin: `${BASE_URL}/auth/login`,
    adminUsers: `${BASE_URL}/admin/users`,
    adminUserById: `${BASE_URL}/admin/users/:userId`,
    updateMe: `${BASE_URL}/users/me`,
    changePassword: `${BASE_URL}/users/me/password`,
} as const;

const devicesHandlers = {
    // POST /devices
    createDeviceHandler: createHttpHandler(
        'post',
        apiEndpoints.devices,
        () => ({
            device: createMockDevice(),
            api_key: 'test_api_key_12345678901234567890123456789012'
        }),
        201 // Created
    ),
    // GET /devices
    getDeviceListHandler: createHttpHandler(
        'get',
        apiEndpoints.devices,
        () => [createMockDevice()]
    ),
    // GET /devices/:deviceId
    getDeviceHandler: createHttpHandler(
        'get',
        apiEndpoints.deviceById,
        (info) => createMockDevice({id: Number(info.params.deviceId)})
    ),
    // DELETE /devices/:deviceId (custom resolver always used; factory unused)
    deleteDeviceHandler: createHttpHandler(
        'delete',
        apiEndpoints.deviceById,
        () => ({}),
        204
    )(undefined, () => responses.noContent()),
}

const addressHandlers = {
    getAddressListHandler: createHttpHandler(
        'get',
        apiEndpoints.deviceAddresses,
        () => [createMockAddress()]
    ),
    createAddressHandler: createHttpHandler(
        'post',
        apiEndpoints.deviceAddresses,
        () => createMockAddress(),
        201
    ),
    heartbeatHandler: createHttpHandler(
        'post',
        apiEndpoints.deviceHeartbeat,
        () => createMockAddress({ip: '192.168.1.100', is_enabled: true}),
        200
    ),
    deleteAddressHandler: createHttpHandler(
        'delete',
        apiEndpoints.deleteDeviceAddresses,
        () => createMockAddress({is_enabled: false}),
        200
    ),
};

const ruleHandlers = {
    getDeviceAddressLeaseRuleHandler: (
        ruleOrNull?: ReturnType<typeof createMockDeviceAddressLeaseRule> | null,
        customResolver?: HttpResponseResolver
    ) => {
        if (customResolver) {
            return http.get(apiEndpoints.deviceAddressLeaseRule, customResolver);
        }
        if (ruleOrNull === undefined || ruleOrNull === null) {
            return http.get(apiEndpoints.deviceAddressLeaseRule, () =>
                responses.notFound({ error: 'Not Found' })
            );
        }
        return http.get(apiEndpoints.deviceAddressLeaseRule, () =>
            HttpResponse.json({
                id: ruleOrNull.id,
                device_id: ruleOrNull.device_id,
                enabled: ruleOrNull.enabled,
                ttl_seconds: ruleOrNull.ttl_seconds,
                created_at:
                    ruleOrNull.created_at instanceof Date
                        ? ruleOrNull.created_at.toISOString()
                        : String(ruleOrNull.created_at),
                updated_at:
                    ruleOrNull.updated_at instanceof Date
                        ? ruleOrNull.updated_at.toISOString()
                        : String(ruleOrNull.updated_at),
            })
        );
    },
    putDeviceAddressLeaseRuleHandler: createHttpHandler(
        'put',
        apiEndpoints.deviceAddressLeaseRule,
        (info) => createMockDeviceAddressLeaseRule({ device_id: Number(info.params.deviceId) }),
        200
    ),
    deleteDeviceAddressLeaseRuleHandler: createHttpHandler(
        'delete',
        apiEndpoints.deviceAddressLeaseRule,
        () => ({}),
        204
    )(undefined, () => responses.noContent()),
};

const authHandlers = {
    meHandler: createHttpHandler(
        'get',
        apiEndpoints.authMe,
        () => createMockUser(),
        200
    ),
    loginHandler: createHttpHandler(
        'post',
        apiEndpoints.authLogin,
        () => createMockUser(),
        200
    ),
    listUsersHandler: createHttpHandler(
        'get',
        apiEndpoints.adminUsers,
        () => [createMockUser()]
    ),
    updateMeHandler: createHttpHandler(
        'patch',
        apiEndpoints.updateMe,
        () => createMockUser()
    ),
    changePasswordHandler: createHttpHandler(
        'post',
        apiEndpoints.changePassword,
        () => ({}),
        204
    )(undefined, () => responses.noContent()),
    adminUpdateUserHandler: createHttpHandler(
        'patch',
        apiEndpoints.adminUserById,
        () => createMockUser()
    ),
    deleteUserHandler: createHttpHandler(
        'delete',
        apiEndpoints.adminUserById,
        () => ({}),
        204
    )(undefined, () => responses.noContent()),
}

export const handlers = {
    devices: {
        ...devicesHandlers
    },
    addresses: {
        ...addressHandlers
    },
    rules: {
        ...ruleHandlers
    },
    auth: {
        ...authHandlers
    }
}


export const responses = {
    // Success (2xx)
    ok: <T extends JsonBodyType>(data: T) =>
        HttpResponse.json(data),

    created: <T extends JsonBodyType>(data: T) =>
        HttpResponse.json(data, {status: 201}),

    noContent: () =>
        new HttpResponse(null, {status: 204}),

    // Client Errors (4xx)
    badRequest: (data: JsonBodyType = {error: 'Bad Request'}) =>
        HttpResponse.json(data, {status: 400}),

    unauthorized: (data: JsonBodyType = {message: 'Unauthorized'}) =>
        HttpResponse.json(data, {status: 401}),

    forbidden: (data: JsonBodyType = {message: 'Forbidden'}) =>
        HttpResponse.json(data, {status: 403}),

    notFound: (data: JsonBodyType = {message: 'Not Found'}) =>
        HttpResponse.json(data, {status: 404}),

    // Server Errors (5xx)
    serverError: (data: JsonBodyType = {message: 'Internal Server Error'}) =>
        HttpResponse.json(data, {status: 500}),

    // Generic Fallback
    custom: (data: JsonBodyType, status: number) =>
        HttpResponse.json(data, {status}),
};

// Overload for array types: dataOverride replaces the entire array
function createHttpHandler<T extends readonly unknown[]>(
    method: keyof typeof http,
    path: string,
    defaultDataFactory: (info: { params: PathParams }) => T,
    defaultStatus?: number
): (
    dataOverride?: T,
    customResolver?: HttpResponseResolver
) => ReturnType<typeof http[keyof typeof http]>;

// Overload for object types: dataOverride merges with defaults
function createHttpHandler<T extends Record<string, unknown>>(
    method: keyof typeof http,
    path: string,
    defaultDataFactory: (info: { params: PathParams }) => T,
    defaultStatus?: number
): (
    dataOverride?: Partial<T>,
    customResolver?: HttpResponseResolver
) => ReturnType<typeof http[keyof typeof http]>;

// Implementation
function createHttpHandler<T extends DefaultBodyType>(
    method: keyof typeof http,
    path: string,
    defaultDataFactory: (info: { params: PathParams }) => T,
    defaultStatus: number = 200
) {
    return (
        dataOverride?: T | Partial<T>,
        customResolver?: HttpResponseResolver
    ) => {
        if (customResolver) {
            return http[method](path, customResolver);
        }

        return http[method](path, (info) => {
            const defaultData = defaultDataFactory(info);

            // 1. Fast path: No overrides? Return default.
            if (dataOverride === undefined) {
                return HttpResponse.json(defaultData, {status: defaultStatus});
            }

            let finalData: T;

            // 2. Handle Lists (Arrays) -> REPLACE, do not merge
            if (Array.isArray(defaultData)) {
                // If it's a list, we assume the override is a complete list replacement
                finalData = dataOverride as T;
            }
            // 3. Handle Entities (Objects) -> MERGE
            else {
                // Type guard: ensure both defaultData and dataOverride are objects before spreading
                if (typeof defaultData === 'object' && defaultData !== null && 
                    typeof dataOverride === 'object' && dataOverride !== null) {
                    const defaultObj = defaultData as Record<string, unknown>;
                    const overrideObj = dataOverride as Record<string, unknown>;
                    finalData = {...defaultObj, ...overrideObj} as T;
                } else {
                    // Fallback: if types don't match, use override as-is or default
                    finalData = (dataOverride ?? defaultData) as T;
                }
            }

            return HttpResponse.json(finalData, {status: defaultStatus});
        });
    }
}
