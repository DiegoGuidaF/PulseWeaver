export function formatEffectiveAccess(item: {
  bypass_host_check: boolean;
  host_count: number;
}): string {
  if (item.bypass_host_check) return "bypass ✱";
  if (item.host_count === 0) return "—";
  return `${item.host_count} ${item.host_count === 1 ? "host" : "hosts"}`;
}
