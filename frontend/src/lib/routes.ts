export const ROUTES = {
  login: "/login",
  dashboard: "/dashboard",
  userDevices: "/user-devices",
  userDeviceWorkspace: "/user-devices/:userId",
  settings: "/settings",
  accessLog: "/access-log",
  addressHistory: "/address-history",
  deviceProvisioning: "/device-provisioning",
  accessHosts: "/access/hosts",
  accessHostGroups: "/access/host-groups",
  accessUsers: "/access/users",
  accessUserDetail: "/access/users/:id",
  policyAudit: "/policy-audit",
  accessNetworkPolicies: "/access/network-policies",
  accessNetworkPolicyDetail: "/access/network-policies/:id",
} as const;

export const buildRoute = {
  userDeviceWorkspace: (userId: string | number) => `/user-devices/${userId}`,
  accessUserDetail: (id: string | number) => `/access/users/${id}`,
  accessNetworkPolicyDetail: (id: string | number) => `/access/network-policies/${id}`,
};
