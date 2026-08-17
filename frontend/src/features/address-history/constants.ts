import { AddressEventKind, AddressEventSource, AddressEventTrigger, TtlRisk } from "@/lib/api";

export const SOURCE_LABELS: Record<AddressEventSource, string> = {
    [AddressEventSource.HEARTBEAT]: "Heartbeat",
    [AddressEventSource.WEB_UI]: "Web UI",
    [AddressEventSource.EXPIRY]: "Expiry",
    [AddressEventSource.LIMIT_EXCEEDED]: "Limit Exceeded",
};

export const SOURCE_OPTIONS = Object.values(AddressEventSource).map((source) => ({
    value: source,
    label: SOURCE_LABELS[source],
}));

/** Narrows a raw URL search-param string to a known address-event source. */
export function isAddressEventSource(value: string): value is AddressEventSource {
    return value in SOURCE_LABELS;
}

export const TRIGGER_LABELS: Record<AddressEventTrigger, string> = {
    [AddressEventTrigger.USER]: "User",
    [AddressEventTrigger.SCHEDULE]: "Scheduled",
    [AddressEventTrigger.NETWORK_CHANGE]: "Network change",
    [AddressEventTrigger.SYSTEM]: "System",
};

export const TRIGGER_OPTIONS = Object.values(AddressEventTrigger).map((trigger) => ({
    value: trigger,
    label: TRIGGER_LABELS[trigger],
}));

/** Narrows a raw URL search-param string to a known address-event trigger. */
export function isAddressEventTrigger(value: string): value is AddressEventTrigger {
    return value in TRIGGER_LABELS;
}

/**
 * Badge colour + emphasis per trigger. `user` is what an admin scans this column
 * for — a deliberate human action — so it takes indigo filled, the style guide's
 * loudest deliberate-action treatment. `network_change` is device-driven
 * liveness and reads one step quieter in amber, while the two background
 * triggers recede to grey: nothing about a routine beat or a server job is worth
 * pulling the eye across the table.
 */
export const TRIGGER_BADGE: Record<AddressEventTrigger, { color: string; variant: "filled" | "light" | "dot" }> = {
    [AddressEventTrigger.USER]: { color: "indigo", variant: "filled" },
    [AddressEventTrigger.NETWORK_CHANGE]: { color: "orange", variant: "light" },
    [AddressEventTrigger.SCHEDULE]: { color: "gray", variant: "dot" },
    [AddressEventTrigger.SYSTEM]: { color: "gray", variant: "dot" },
};

export const EVENT_KIND_LABELS: Record<AddressEventKind, string> = {
    [AddressEventKind.CREATED]: "Created",
    [AddressEventKind.ENABLED]: "Enabled",
    [AddressEventKind.DISABLED]: "Disabled",
    [AddressEventKind.REFRESH]: "Refresh",
};

export const EVENT_KIND_OPTIONS = Object.values(AddressEventKind).map((kind) => ({
    value: kind,
    label: EVENT_KIND_LABELS[kind],
}));

export const EVENT_KIND_COLORS: Record<AddressEventKind, string> = {
    [AddressEventKind.CREATED]: "indigo",
    [AddressEventKind.ENABLED]: "green",
    [AddressEventKind.DISABLED]: "red",
    [AddressEventKind.REFRESH]: "gray",
};

/**
 * The `ttl_risk` enum measures how much of a device's lease TTL its renewal
 * consumed, so it is surfaced as "TTL headroom" — a quantity with a direction,
 * not a verdict. A renewal past its TTL is the routine result of a device
 * sleeping, so the labels rank headroom rather than naming failures: an admin
 * reading this screen is sizing a TTL, not triaging an incident.
 * `unknown` is not a level but an absence of one, so it reads as
 * "Not applicable" in pickers and as a dimmed dash in the table.
 */
export const TTL_HEADROOM_COLUMN_LABEL = "TTL headroom";

export const TTL_RISK_LABELS: Record<TtlRisk, string> = {
    [TtlRisk.UNKNOWN]: "Not applicable",
    [TtlRisk.OK]: "Comfortable",
    [TtlRisk.APPROACHING]: "Tight",
    [TtlRisk.CRITICAL]: "Very tight",
    [TtlRisk.BREACHED]: "Past TTL",
};

/** Why a row carries no headroom reading, shown on the dash that replaces the badge. */
export const TTL_RISK_UNKNOWN_HINT =
    "No TTL headroom to report — this event does not renew a lease, or the device has no lease rule.";

export const TTL_RISK_OPTIONS = Object.values(TtlRisk).map((risk) => ({
    value: risk,
    label: TTL_RISK_LABELS[risk],
}));

/**
 * Badge color + emphasis per headroom level: indigo for the comfortable norm,
 * then an amber ramp deepening as headroom runs out. Red is deliberately absent
 * — it is reserved for states that actually cost a user access, and a renewal
 * arriving after its lease expired is a tuning signal, not an outage.
 */
export const TTL_RISK_BADGE: Record<TtlRisk, { color: string; variant: "light" | "filled" }> = {
    [TtlRisk.UNKNOWN]: { color: "gray", variant: "light" },
    [TtlRisk.OK]: { color: "indigo", variant: "light" },
    [TtlRisk.APPROACHING]: { color: "orange", variant: "light" },
    [TtlRisk.CRITICAL]: { color: "orange", variant: "filled" },
    [TtlRisk.BREACHED]: { color: "orange.9", variant: "filled" },
};

/**
 * Chart bands, bottom of the stack first. `approaching` and `critical` share
 * one band: four one-hue steps cannot hold adjacent-pair colour separation
 * (measured — the four-step ramp lands at ΔE 8.8 against a floor of 15, while
 * these three clear it at 22), and the 0.7–0.9 vs 0.9–1.0 split is finer than a
 * stacked bar can express anyway. The table column still ranks all four levels.
 */
export const RENEWAL_TIMING_BANDS = [
    { label: TTL_RISK_LABELS[TtlRisk.OK], color: "indigo.4" },
    { label: "Near TTL", color: "orange.4" },
    { label: TTL_RISK_LABELS[TtlRisk.BREACHED], color: "orange.9" },
] as const;

/** Per-band device counts for one bucket, in `RENEWAL_TIMING_BANDS` order. */
export function renewalTimingBandCounts(bucket: {
    ok_device_count: number;
    approaching_device_count: number;
    critical_device_count: number;
    breached_device_count: number;
}): Record<string, number> {
    return {
        [RENEWAL_TIMING_BANDS[0].label]: bucket.ok_device_count,
        [RENEWAL_TIMING_BANDS[1].label]: bucket.approaching_device_count + bucket.critical_device_count,
        [RENEWAL_TIMING_BANDS[2].label]: bucket.breached_device_count,
    };
}

/** Formats a duration in seconds as a compact human string, e.g. "45s", "12m", "1h 30m", "2d 4h". */
export function formatGapDuration(seconds: number): string {
    if (seconds < 60) return `${seconds}s`;

    const minutes = Math.round(seconds / 60);
    if (minutes < 60) return `${minutes}m`;

    const hours = Math.floor(minutes / 60);
    const remMinutes = minutes % 60;
    if (hours < 24) return remMinutes > 0 ? `${hours}h ${remMinutes}m` : `${hours}h`;

    const days = Math.floor(hours / 24);
    const remHours = hours % 24;
    return remHours > 0 ? `${days}d ${remHours}h` : `${days}d`;
}
