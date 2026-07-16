import { DevicePairingStatus, PairingExpiryHours } from "@/lib/api";

export const PAIRING_STATUS_BADGE: Record<DevicePairingStatus, { label: string; color: string }> = {
  [DevicePairingStatus.PENDING]: { label: "pending", color: "indigo" },
  [DevicePairingStatus.USED]: { label: "claimed", color: "green" },
  [DevicePairingStatus.EXPIRED]: { label: "expired", color: "red" },
  [DevicePairingStatus.INVALIDATED]: { label: "revoked", color: "gray" },
  [DevicePairingStatus.REPLACED]: { label: "replaced", color: "gray" },
};

export const PAIRING_EXPIRY_LABELS: Record<PairingExpiryHours, string> = {
  [PairingExpiryHours[1]]: "1 hour",
  [PairingExpiryHours[24]]: "24 hours",
  [PairingExpiryHours[48]]: "48 hours",
  [PairingExpiryHours[168]]: "7 days",
};

export const PAIRING_EXPIRY_OPTIONS = Object.values(PairingExpiryHours).map((hours) => ({
  value: String(hours),
  label: PAIRING_EXPIRY_LABELS[hours],
}));
