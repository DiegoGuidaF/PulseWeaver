/**
 * The address-lease TTL ladder, shared by the control that sets a device's TTL
 * and by the surfaces that suggest a new one. A suggestion must land on a value
 * the control actually offers, so both sides read this one list.
 */
export const TTL_PRESET_SECONDS = [30, 300, 900, 3600, 21600, 86400] as const;

const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_HOUR = 3600;
const SECONDS_PER_DAY = 86400;

/** Compact TTL label, e.g. "30s", "5m", "1h", "24h". */
export function formatTtlLabel(ttlSeconds: number): string {
    if (ttlSeconds < SECONDS_PER_MINUTE) return `${ttlSeconds}s`;
    if (ttlSeconds < SECONDS_PER_HOUR) return `${Math.round(ttlSeconds / SECONDS_PER_MINUTE)}m`;
    if (ttlSeconds < SECONDS_PER_DAY) return `${Math.round(ttlSeconds / SECONDS_PER_HOUR)}h`;
    return `${Math.round(ttlSeconds / SECONDS_PER_DAY)}d`;
}

/**
 * The lowest preset that would have covered `gapSeconds` — the TTL to move a
 * device to so renewals of that size stop landing past their lease. Returns
 * `null` when the gap exceeds the top of the ladder, which needs a custom value
 * rather than a preset.
 */
export function suggestTtlSeconds(gapSeconds: number): number | null {
    return TTL_PRESET_SECONDS.find((preset) => preset >= gapSeconds) ?? null;
}
