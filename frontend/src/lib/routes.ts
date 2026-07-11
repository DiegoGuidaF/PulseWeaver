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
  anomalies: "/anomalies",
} as const;

export const buildRoute = {
  userDevices: (ownerId: string | number) => `/devices/owners/${ownerId}`,
  userDevicesNew: (ownerId: string | number) => `/devices/owners/${ownerId}/new`,
  accessUserDetail: (id: string | number) => `/access/users/${id}`,
  accessNetworkPolicyDetail: (id: string | number) => `/access/network-policies/${id}`,
};
