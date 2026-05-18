import type { NetworkPolicySummary } from "@/lib/api";

export const CIDR_RE = /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\/\d{1,2}$/;

export function networkPolicyHostAccessSummary(
    policy: Pick<NetworkPolicySummary, "allow_all_hosts" | "effective_host_count" | "total_host_count">,
): string {
    if (policy.allow_all_hosts) return "All hosts";
    if (policy.effective_host_count === 0) return "—";
    return `${policy.effective_host_count} / ${policy.total_host_count}`;
}
