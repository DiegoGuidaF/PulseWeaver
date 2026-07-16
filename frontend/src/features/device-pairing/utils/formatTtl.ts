import dayjs from "dayjs";

/** Renders the time remaining until `expiresAt`, e.g. "45m remaining" or "2h 15m remaining". */
export function formatTtl(expiresAt: string): string {
  const diffMin = dayjs(expiresAt).diff(dayjs(), "minute");
  if (diffMin <= 0) return "expired";
  if (diffMin < 60) return `${diffMin}m remaining`;
  const h = Math.floor(diffMin / 60);
  const m = diffMin % 60;
  return m > 0 ? `${h}h ${m}m remaining` : `${h}h remaining`;
}
