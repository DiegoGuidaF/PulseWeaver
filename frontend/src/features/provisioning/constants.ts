import type { PendingRegistration } from "@/lib/api";

export type FilterTab = "pending" | "used" | "expired" | "all";

export const FILTER_TAB_OPTIONS: { value: FilterTab; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "used", label: "Used" },
  { value: "expired", label: "Expired" },
  { value: "all", label: "All" },
];

export const STATUS_BADGE: Record<
  PendingRegistration["status"],
  { color: string; label: string }
> = {
  pending: { color: "green", label: "Pending" },
  used: { color: "gray", label: "Used" },
  expired: { color: "red", label: "Expired" },
};

export const EXPIRING_SOON_MS = 60 * 60 * 1000;
