export const queryKeys = {
  devices: {
    all: ["devices"] as const,
    detail: (id: number) => ["devices", id] as const,
    addresses: (deviceId: number) => ["device-addresses", deviceId] as const,
  },
  auth: {
    currentUser: ["auth", "currentUser"] as const,
  },
};
