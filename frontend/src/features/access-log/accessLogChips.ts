import type { FilterChip } from "@/components/ActiveFilterChips";
import {
    type FilterColumnKey,
    COLUMN_CHIP_LABELS,
    FILTER_COLUMN_KEYS,
    describeColumnFilter,
    isFilterActive,
} from "./filterConfig";
import type { AccessLogFilters } from "./hooks/useAccessLogFilters";

/** Maps a stored filter value (often an id) to the name the chip should show. */
export type ChipValueLabels = Partial<Record<FilterColumnKey, (value: string) => string>>;

interface BuildChipsOptions {
    filters: AccessLogFilters;
    formatDateTime: (value: string) => string;
    valueLabels: ChipValueLabels;
}

/**
 * The removable chip row summarising every active filter: the time window and
 * the outcome first, then one chip per active column filter in column order.
 *
 * An open-ended window still gets a chip — "→ now" is a filter the user set and
 * must be able to lift, so only a completely unset from/to is skipped.
 */
export function buildAccessLogChips({
    filters,
    formatDateTime,
    valueLabels,
}: BuildChipsOptions): FilterChip[] {
    const chips: FilterChip[] = [];

    if (filters.fromStr || filters.toStr) {
        chips.push({
            label: "Time",
            value: `${filters.fromStr ? formatDateTime(filters.fromStr) : "—"} → ${
                filters.toStr ? formatDateTime(filters.toStr) : "now"
            }`,
            onRemove: () =>
                filters.setSearchParams((prev) => {
                    prev.delete("from");
                    prev.delete("to");
                    return prev;
                }),
        });
    }

    if (filters.outcomeStr) {
        chips.push({
            label: "Outcome",
            value: filters.outcomeStr === "allow" ? "Allow" : "Deny",
            onRemove: () => filters.setOutcome(null),
        });
    }

    for (const key of FILTER_COLUMN_KEYS) {
        const state = filters.getColumnFilter(key);
        if (!isFilterActive(state)) continue;
        chips.push({
            label: COLUMN_CHIP_LABELS[key],
            value: describeColumnFilter(key, state, valueLabels[key]),
            onRemove: () => filters.setColumnFilter(key, null),
        });
    }

    return chips;
}
