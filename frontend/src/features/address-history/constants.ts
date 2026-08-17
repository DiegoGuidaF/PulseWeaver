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
 * Keyed only by the trigger that earns a badge; every other trigger renders as
 * plain text. `user` means a human had to press the button to get the device
 * back online — the signal this axis exists for — and indigo is the style
 * guide's deliberate-action colour. The background triggers are ~93% of the
 * column, and colour spent on a column's usual value says nothing.
 */
export const TRIGGER_BADGE: Partial<Record<AddressEventTrigger, { color: string; variant: "filled" }>> = {
    [AddressEventTrigger.USER]: { color: "indigo", variant: "filled" },
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

/**
 * Keyed only by the kinds that earn a badge. `created` and `enabled` share green
 * because they are one fact — the address is allowed through now — and the label
 * carries which of the two; `disabled` is red as the only state on this screen
 * that costs someone access. `refresh` is 86% of all events, so it renders as
 * plain text: it is the background the other three have to stand out against.
 */
export const EVENT_KIND_BADGE_COLORS: Partial<Record<AddressEventKind, string>> = {
    [AddressEventKind.CREATED]: "green",
    [AddressEventKind.ENABLED]: "green",
    [AddressEventKind.DISABLED]: "red",
};

/**
 * The address state each kind implies, so the cell can state the state only when
 * it does not follow. It usually does: measured over 7,071 real events, the
 * implication holds on 7,069.
 *
 * Both loopholes are real. `created` means "no earlier event for this address
 * survives", not "the address was created" — the retention job prunes
 * `address_events`, so the oldest surviving event can be a cap eviction on an
 * address that is disabled. And `refresh` only means "state unchanged", so
 * disabling an already-disabled address writes a second disabled event that
 * classifies as a refresh.
 */
export const EVENT_KIND_IMPLIES_ENABLED: Record<AddressEventKind, boolean> = {
    [AddressEventKind.CREATED]: true,
    [AddressEventKind.ENABLED]: true,
    [AddressEventKind.DISABLED]: false,
    [AddressEventKind.REFRESH]: true,
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
 * Keyed only by the levels that earn a badge: an amber ramp deepening as
 * headroom runs out. `ok` is 88% of rows and renders as plain text, `unknown` as
 * a dash — neither is a finding, and a badge on the comfortable norm is what
 * made this column unreadable.
 *
 * The amber is the style guide's *liveness* amber, not its warning colour: this
 * ramp measures heartbeat cadence against lease length, which is the axis amber
 * is reserved for. Red is deliberately absent — it belongs to states that cost a
 * user access, and a renewal arriving after its lease expired is a tuning
 * signal, not an outage.
 *
 * The two filled steps carry black text and a pinned shade. Mantine's `filled`
 * pairs white with whichever shade the scheme picks, which measures 3.04 and
 * 4.30 in light and 3.58 in dark — under the 4.5:1 floor these 10px bold labels
 * need. Black on the same amber measures 8.17 and 4.88, and pinning the shade
 * stops the ramp shifting a step between schemes.
 */
export const TTL_RISK_BADGE: Partial<
    Record<TtlRisk, { color: string; variant: "light" | "filled"; textColor?: string }>
> = {
    [TtlRisk.APPROACHING]: { color: "orange", variant: "light" },
    [TtlRisk.CRITICAL]: { color: "orange.6", variant: "filled", textColor: "black" },
    [TtlRisk.BREACHED]: { color: "orange.9", variant: "filled", textColor: "black" },
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
