import { AddressHistoryFilterOperator } from "@/lib/api";
import { LEASE_HEALTH_COLUMN_LABEL } from "./constants";
import {
    isFilterActive,
    operatorLabel,
    type ColumnFilterState as BaseColumnFilterState,
    type FilterColumnConfig,
} from "@/lib/columnFilter";

export type FilterOp = AddressHistoryFilterOperator;
export type ColumnFilterState = BaseColumnFilterState<FilterOp>;

export type FilterColumnKey = "device_id" | "user_id" | "ip" | "source" | "event_kind" | "ttl_risk";

export const FILTER_COLUMNS: Record<FilterColumnKey, FilterColumnConfig<FilterOp>> = {
    device_id: {
        operators: [AddressHistoryFilterOperator.IN, AddressHistoryFilterOperator.NOT_IN],
        widget: "multiselect",
        numeric: true,
    },
    user_id: {
        operators: [AddressHistoryFilterOperator.IN, AddressHistoryFilterOperator.NOT_IN],
        widget: "multiselect",
        numeric: true,
    },
    ip: {
        operators: [
            AddressHistoryFilterOperator.IN,
            AddressHistoryFilterOperator.NOT_IN,
            AddressHistoryFilterOperator.CONTAINS,
            AddressHistoryFilterOperator.NOT_CONTAINS,
        ],
        widget: "tags",
    },
    source: {
        operators: [AddressHistoryFilterOperator.IN, AddressHistoryFilterOperator.NOT_IN],
        widget: "multiselect",
    },
    event_kind: {
        operators: [AddressHistoryFilterOperator.IN, AddressHistoryFilterOperator.NOT_IN],
        widget: "multiselect",
    },
    ttl_risk: {
        operators: [AddressHistoryFilterOperator.IN, AddressHistoryFilterOperator.NOT_IN],
        widget: "multiselect",
    },
};

export const FILTER_COLUMN_KEYS = Object.keys(FILTER_COLUMNS) as FilterColumnKey[];

export { isFilterActive, operatorLabel };

/** Chip labels for each filter column. */
export const COLUMN_CHIP_LABELS: Record<FilterColumnKey, string> = {
    device_id: "Device",
    user_id: "User",
    ip: "IP",
    source: "Source",
    event_kind: "Event",
    ttl_risk: LEASE_HEALTH_COLUMN_LABEL,
};

/**
 * Renders a column filter as a chip value, e.g. "is any of Heartbeat, Manual".
 * `resolveLabel` maps stored values (often enum values or IDs) to display names.
 */
export function describeColumnFilter(
    key: FilterColumnKey,
    state: ColumnFilterState,
    resolveLabel?: (value: string) => string,
): string {
    const config = FILTER_COLUMNS[key];
    const label = operatorLabel(config, state.op);
    const values = resolveLabel ? state.values.map(resolveLabel) : state.values;
    return `${label} ${values.join(", ")}`;
}
