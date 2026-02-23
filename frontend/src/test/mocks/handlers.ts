import {
    type DefaultBodyType,
    http,
    HttpResponse,
    type HttpResponseResolver,
    type JsonBodyType,
    type PathParams
} from "msw";
import {createMockAddress, createMockDevice, createMockUser} from "@/test/mocks/data.ts";

const BASE_URL = '/api/v1';

const apiEndpoints = {
    devices: `${BASE_URL}/devices`,
    deviceById: `${BASE_URL}/devices/:deviceId`,
    deviceAddresses: `${BASE_URL}/devices/:deviceId/addresses`,
    deleteDeviceAddresses: `${BASE_URL}/devices/:deviceId/addresses/:addressId`,
    authMe: `${BASE_URL}/auth/me`,
    authLogin: `${BASE_URL}/auth/login`,
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
    deleteAddressHandler: createHttpHandler(
        'delete',
        apiEndpoints.deleteDeviceAddresses,
        () => createMockAddress({status: false}),
        200
    ),
}

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
    )
}

export const handlers = {
    devices: {
        ...devicesHandlers
    },
    addresses: {
        ...addressHandlers
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
