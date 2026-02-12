export const queryKeys = {
  auth: {
    currentUser: ["auth", "currentUser"] as const,
  },
  devices: {
    all: ["devices"] as const,
    detail: (id: number) => ["devices", id] as const,
    addresses: (deviceId: number) =>
      ["device-addresses", deviceId] as const,
  },
} as const;
