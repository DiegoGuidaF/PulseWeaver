import type { PolicyUserEntry } from "@/lib/api";

/**
 * Two-dimensional status derived from a user's policy entry.
 *
 * The five statuses map two orthogonal facts:
 *   - reachability: does the user currently have live IPs in the cache?
 *   - authorization: does the user have any host grants (or bypass)?
 *
 * "bypass"            — bypass flag is on; host check is skipped entirely
 * "live_with_access"  — live IPs present AND host grants exist
 * "live_no_host_access" — live IPs present BUT no host grants and bypass is off
 *                         (device is active but all requests will be denied)
 * "no_live_ips"       — no live IPs in the cache, but host grants exist
 * "no_access"         — no live IPs AND no host grants AND bypass is off
 */
export type UserStatus =
  | "bypass"
  | "live_with_access"
  | "live_no_host_access"
  | "no_live_ips"
  | "no_access";

export function deriveUserStatus(user: PolicyUserEntry): UserStatus {
  if (user.bypass_allowlist) return "bypass";
  const hasLiveIps = user.ips.length > 0;
  const hasHostGrants = user.allowed_host_count > 0;
  if (hasLiveIps && hasHostGrants) return "live_with_access";
  if (hasLiveIps && !hasHostGrants) return "live_no_host_access";
  if (!hasLiveIps && hasHostGrants) return "no_live_ips";
  return "no_access";
}
