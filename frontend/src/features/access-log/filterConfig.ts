import { AccessLogSortColumn, SortOrder, type AccessLogFilterOperator } from "@/lib/api";
import {
    isFilterActive,
    operatorLabel,
    type ColumnFilterState as BaseColumnFilterState,
    type FilterColumnConfig,
} from "@/lib/columnFilter";

export type FilterOp = AccessLogFilterOperator;
export type ColumnFilterState = BaseColumnFilterState<FilterOp>;

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

export const FILTER_COLUMNS: Record<FilterColumnKey, FilterColumnConfig<FilterOp>> = {
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

export { isFilterActive, operatorLabel };

/**
 * Static HTTP-method options. The backend has no distinct-values endpoint yet.
 * TODO(PW-24): replace with a backend distinct-values query (same future
 * treatment applies to `country_code` and `user_id` option sources).
 */
export const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

export type SortColumn = AccessLogSortColumn;

export type SortDirection = SortOrder;

/** Resting sort, equivalent to no `sort`/`order` params: newest first. */
export const DEFAULT_SORT: SortColumn = AccessLogSortColumn.CREATED_AT;
export const DEFAULT_ORDER: SortDirection = SortOrder.DESC;

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
        return { sort: clicked, order: clicked === DEFAULT_SORT ? SortOrder.DESC : SortOrder.ASC };
    }
    if (clicked === DEFAULT_SORT) {
        return { sort: DEFAULT_SORT, order: current.order === SortOrder.DESC ? SortOrder.ASC : SortOrder.DESC };
    }
    if (current.order === SortOrder.ASC) return { sort: clicked, order: SortOrder.DESC };
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
