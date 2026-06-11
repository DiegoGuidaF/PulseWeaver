import type { AccessLogFilterOperator } from "@/lib/api";

export type FilterOp = AccessLogFilterOperator;

/** A column's filter as held in the URL: an operator plus 0..N values. */
export interface ColumnFilterState {
    op: FilterOp;
    values: string[];
}

export type FilterColumnKey =
    | "client_ip"
    | "target_host"
    | "target_uri"
    | "http_method"
    | "deny_reason"
    | "country_code"
    | "device_id"
    | "user_id"
    | "network_policy_id";

/** Value widget used for the multi-value operators (`in` / `not_in`). */
export type ValueWidget = "tags" | "multiselect";

export interface FilterColumnConfig {
    operators: FilterOp[];
    widget: ValueWidget;
    /** Whether the column's values are numeric IDs (device/user/policy). */
    numeric?: boolean;
    /** Column-tuned wording for the value-less operators. */
    emptyLabel?: string;
    notEmptyLabel?: string;
}

export const FILTER_COLUMNS: Record<FilterColumnKey, FilterColumnConfig> = {
    client_ip: {
        operators: ["in", "not_in", "contains", "not_contains"],
        widget: "tags",
    },
    target_host: {
        operators: ["in", "not_in", "contains", "not_contains", "is_null", "not_null"],
        widget: "tags",
        emptyLabel: "has none",
    },
    target_uri: {
        operators: ["in", "not_in", "contains", "not_contains", "is_null", "not_null"],
        widget: "tags",
        emptyLabel: "has none",
    },
    http_method: {
        operators: ["in", "not_in"],
        widget: "multiselect",
    },
    deny_reason: {
        operators: ["in", "not_in", "is_null", "not_null"],
        widget: "multiselect",
    },
    country_code: {
        operators: ["in", "not_in", "is_null", "not_null"],
        widget: "tags",
        emptyLabel: "is unknown",
    },
    device_id: {
        operators: ["in", "not_in", "is_null", "not_null"],
        widget: "multiselect",
        numeric: true,
    },
    user_id: {
        operators: ["in", "not_in"],
        widget: "multiselect",
        numeric: true,
    },
    network_policy_id: {
        operators: ["in", "not_in", "is_null", "not_null"],
        widget: "multiselect",
        numeric: true,
    },
};

export const FILTER_COLUMN_KEYS = Object.keys(FILTER_COLUMNS) as FilterColumnKey[];

export const OP_LABELS: Record<FilterOp, string> = {
    in: "is any of",
    not_in: "is none of",
    contains: "contains",
    not_contains: "does not contain",
    is_null: "is empty",
    not_null: "is not empty",
};

/** Human label for an operator, honouring a column's empty/not-empty overrides. */
export function operatorLabel(config: FilterColumnConfig, op: FilterOp): string {
    if (op === "is_null" && config.emptyLabel) return config.emptyLabel;
    if (op === "not_null" && config.notEmptyLabel) return config.notEmptyLabel;
    return OP_LABELS[op];
}

/** A column filter is active when it has values, or uses a value-less operator. */
export function isFilterActive(state: ColumnFilterState): boolean {
    return state.values.length > 0 || state.op === "is_null" || state.op === "not_null";
}

/**
 * Static HTTP-method options. The backend has no distinct-values endpoint yet.
 * TODO(PW-24): replace with a backend distinct-values query (same future
 * treatment applies to `country_code` and `user_id` option sources).
 */
export const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

export const SORTABLE_COLUMNS = [
    "created_at",
    "client_ip",
    "target_host",
    "http_method",
    "country_code",
    "deny_reason",
    "duration_us",
    "outcome",
] as const;

export type SortColumn = (typeof SORTABLE_COLUMNS)[number];

export type SortDirection = "asc" | "desc";

/** Resting sort, equivalent to no `sort`/`order` params: newest first. */
export const DEFAULT_SORT: SortColumn = "created_at";
export const DEFAULT_ORDER: SortDirection = "desc";

export interface SortState {
    sort: SortColumn;
    order: SortDirection;
}

/**
 * Advances the sort one step when a header is clicked. A non-default column
 * cycles asc → desc → cleared (back to the default newest-first); the default
 * column has no distinct cleared state — its desc *is* the baseline — so it
 * just toggles asc ⇄ desc.
 */
export function nextSortState(current: SortState, clicked: SortColumn): SortState {
    if (clicked !== current.sort) {
        return { sort: clicked, order: clicked === DEFAULT_SORT ? "desc" : "asc" };
    }
    if (clicked === DEFAULT_SORT) {
        return { sort: DEFAULT_SORT, order: current.order === "desc" ? "asc" : "desc" };
    }
    if (current.order === "asc") return { sort: clicked, order: "desc" };
    return { sort: DEFAULT_SORT, order: DEFAULT_ORDER };
}

/** Chip labels for each filter column. */
export const COLUMN_CHIP_LABELS: Record<FilterColumnKey, string> = {
    client_ip: "IP",
    target_host: "Host",
    target_uri: "URI",
    http_method: "Method",
    deny_reason: "Reason",
    country_code: "Country",
    device_id: "Device",
    user_id: "User",
    network_policy_id: "Network policy",
};

/**
 * Renders a column filter as a chip value, e.g. "is any of DE, US" or
 * "is unknown". `resolveLabel` maps stored values (often IDs) to display names.
 */
export function describeColumnFilter(
    key: FilterColumnKey,
    state: ColumnFilterState,
    resolveLabel?: (value: string) => string,
): string {
    const config = FILTER_COLUMNS[key];
    const label = operatorLabel(config, state.op);
    if (state.op === "is_null" || state.op === "not_null") return label;
    const values = resolveLabel ? state.values.map(resolveLabel) : state.values;
    return `${label} ${values.join(", ")}`;
}
