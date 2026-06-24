import { PolicyDenyReason, type PolicyDenyReason as PolicyDenyReasonValue } from "@/lib/api";

export const POLICY_DENY_REASON_LABELS: Record<PolicyDenyReasonValue, string> = {
  no_device_match: "No matching device",
  ip_not_registered: "IP not in any device or network policy",
  invalid_token: "Invalid token",
  host_not_allowed: "Host not allowed",
};

export const POLICY_DENY_REASON_OPTIONS: { value: PolicyDenyReasonValue; label: string }[] = Object.values(
  PolicyDenyReason,
).map((reason) => ({
  value: reason,
  label: POLICY_DENY_REASON_LABELS[reason],
}));
