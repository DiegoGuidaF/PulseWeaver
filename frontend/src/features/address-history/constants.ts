import { AddressEventKind, AddressEventSource, TtlRisk } from "@/lib/api";
import type { ColumnFilterState } from "@/lib/columnFilter";

export const SOURCE_LABELS: Record<AddressEventSource, string> = {
    [AddressEventSource.HEARTBEAT]: "Heartbeat",
    [AddressEventSource.MANUAL]: "Manual",
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
 * Event kinds that represent a real address state change. The default "state
 * changes" view filters to these, excluding routine heartbeat refreshes.
 */
export const CHANGE_EVENT_KINDS: AddressEventKind[] = [
    AddressEventKind.CREATED,
    AddressEventKind.ENABLED,
    AddressEventKind.DISABLED,
];

/** True when a column filter matches exactly the default "state changes" event-kind set. */
export function isStateChangesOnly(state: ColumnFilterState): boolean {
    return (
        state.op === "in" &&
        state.values.length === CHANGE_EVENT_KINDS.length &&
        CHANGE_EVENT_KINDS.every((k) => state.values.includes(k))
    );
}

export const TTL_RISK_LABELS: Record<TtlRisk, string> = {
    [TtlRisk.UNKNOWN]: "Unknown",
    [TtlRisk.OK]: "OK",
    [TtlRisk.APPROACHING]: "Approaching",
    [TtlRisk.CRITICAL]: "Critical",
    [TtlRisk.BREACHED]: "Breached",
};

export const TTL_RISK_OPTIONS = Object.values(TtlRisk).map((risk) => ({
    value: risk,
    label: TTL_RISK_LABELS[risk],
}));

/** Badge color + emphasis per risk level, escalating from neutral to filled red. */
export const TTL_RISK_BADGE: Record<TtlRisk, { color: string; variant: "light" | "filled" }> = {
    [TtlRisk.UNKNOWN]: { color: "gray", variant: "light" },
    [TtlRisk.OK]: { color: "green", variant: "light" },
    [TtlRisk.APPROACHING]: { color: "yellow", variant: "light" },
    [TtlRisk.CRITICAL]: { color: "red", variant: "light" },
    [TtlRisk.BREACHED]: { color: "red", variant: "filled" },
};

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
