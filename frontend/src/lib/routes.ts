import { SUGGESTED_TTL_PARAM } from "@/lib/ttlPresets";

export const ROUTES = {
  login: "/login",
  dashboard: "/dashboard",
  devices: "/devices",
  userDevices: "/devices/owners/:ownerId",
  userDevicesNew: "/devices/owners/:ownerId/new",
  account: "/account",
  accessLog: "/access-log",
  addressHistory: "/address-history",
  accessHosts: "/access/hosts",
  accessHostGroups: "/access/host-groups",
  accessUsers: "/access/users",
  accessUserDetail: "/access/users/:id",
  policyAudit: "/policy-audit",
  accessNetworkPolicies: "/access/network-policies",
  accessNetworkPolicyDetail: "/access/network-policies/:id",
} as const;

/**
 * Tab vocabulary for the device workspace's `?tab=` param, shared by the page
 * that reads it and the deep links that write it so the two cannot drift.
 */
export const DeviceTab = {
  ADDRESSES: "addresses",
  RULES: "rules",
  PAIRING: "pairing",
  HISTORY: "history",
  SETTINGS: "settings",
} as const;

export type DeviceTabValue = (typeof DeviceTab)[keyof typeof DeviceTab];

/**
 * Resolves the `?returnTo=` login param to an in-app path. Only a single-slash
 * absolute path is accepted: `//host`, `/\host`, `https://…` and control
 * characters can be read as another origin by a browser, so they fall back to
 * the dashboard instead of being handed to the router.
 */
export function safeReturnTo(raw: string | null): string {
  if (!raw || !/^\/(?![/\\])/.test(raw)) return ROUTES.dashboard;
  const hasControlChar = [...raw].some((c) => c.charCodeAt(0) <= 0x1f || c.charCodeAt(0) === 0x7f);
  return hasControlChar ? ROUTES.dashboard : raw;
}

export const buildRoute = {
  userDevices: (ownerId: string | number) => `/devices/owners/${ownerId}`,
  userDevicesNew: (ownerId: string | number) => `/devices/owners/${ownerId}/new`,
  /**
   * The device's Rules tab. `suggestedTtlSeconds` stages a TTL on the
   * auto-expiry control for the user to confirm — it is never applied by
   * following the link.
   */
  deviceRules: (ownerId: string | number, deviceId: string | number, suggestedTtlSeconds?: number) => {
    const params = new URLSearchParams({ device: String(deviceId), tab: DeviceTab.RULES });
    if (suggestedTtlSeconds !== undefined) params.set(SUGGESTED_TTL_PARAM, String(suggestedTtlSeconds));
    return `/devices/owners/${ownerId}?${params}`;
  },
  accessUserDetail: (id: string | number) => `/access/users/${id}`,
  accessNetworkPolicyDetail: (id: string | number) => `/access/network-policies/${id}`,
};
