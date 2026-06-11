import { useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import dayjs from "dayjs";
import type { GetAccessLogData } from "@/lib/api";
import { DEFAULT_PRESET_KEY, PRESET_MS } from "../constants";
import {
    type ColumnFilterState,
    type FilterColumnKey,
    type FilterOp,
    type SortColumn,
    FILTER_COLUMN_KEYS,
    FILTER_COLUMNS,
    isFilterActive,
} from "../filterConfig";

type Query = NonNullable<GetAccessLogData["query"]>;

const LS_KEY = "pulseweaver:access-log:filters";
const DEFAULT_PARAMS = new URLSearchParams({ preset: DEFAULT_PRESET_KEY });

/** Save current search params to localStorage for next visit. */
function persistFilters(params: URLSearchParams) {
    const str = params.toString();
    if (str) localStorage.setItem(LS_KEY, str);
    else localStorage.removeItem(LS_KEY);
}

export interface AccessLogFilters {
    queryParams: Query;
    filterKey: string;

    // Time window
    presetStr: string | null;
    fromStr: string | null;
    toStr: string | null;

    // Outcome (allow/deny) — kept as a dedicated boolean filter, not a column op
    outcomeStr: string | null;

    // Sort
    sort: SortColumn;
    order: "asc" | "desc";

    /** True when a custom `to` is set (historical view, not live tail). */
    hasCustomTo: boolean;
    /** True when any value/outcome/time filter is active. */
    hasActiveFilters: boolean;

    // Column filter accessors (operator + values)
    getColumnFilter: (key: FilterColumnKey) => ColumnFilterState;
    setColumnFilter: (key: FilterColumnKey, state: ColumnFilterState | null) => void;

    // Setters
    setPreset: (key: string | null) => void;
    setOutcome: (value: string | null) => void;
    setSort: (column: SortColumn, order: "asc" | "desc") => void;
    setSearchParams: (updater: URLSearchParams | ((prev: URLSearchParams) => URLSearchParams)) => void;
    clearAll: () => void;
}

function getInitialParams(): URLSearchParams {
    const init = new URLSearchParams(window.location.search);
    if (init.toString() === "") {
        const saved = localStorage.getItem(LS_KEY);
        if (saved) return new URLSearchParams(saved);
        return new URLSearchParams(DEFAULT_PARAMS);
    }
    return init;
}

export function useAccessLogFilters(): AccessLogFilters {
    const [searchParams, setSearchParamsRaw] = useSearchParams(getInitialParams());

    // Wrap setSearchParams to persist every change to localStorage
    const setSearchParams: AccessLogFilters["setSearchParams"] = useCallback(
        (updater) => {
            setSearchParamsRaw((prev) => {
                const next = typeof updater === "function" ? updater(prev) : updater;
                persistFilters(next);
                return next;
            });
        },
        [setSearchParamsRaw],
    );

    const presetStr = searchParams.get("preset");
    const fromStr = searchParams.get("from");
    const toStr = searchParams.get("to");
    const outcomeStr = searchParams.get("outcome");
    const sort = (searchParams.get("sort") as SortColumn | null) ?? "created_at";
    const order = (searchParams.get("order") as "asc" | "desc" | null) ?? "desc";

    const getColumnFilter = useCallback(
        (key: FilterColumnKey): ColumnFilterState => ({
            op: (searchParams.get(`${key}_op`) as FilterOp | null) ?? "in",
            values: searchParams.getAll(key),
        }),
        [searchParams],
    );

    const setColumnFilter = useCallback(
        (key: FilterColumnKey, state: ColumnFilterState | null) => {
            setSearchParams((prev) => {
                prev.delete(key);
                prev.delete(`${key}_op`);
                if (state) {
                    const isNullOp = state.op === "is_null" || state.op === "not_null";
                    if (isNullOp) {
                        prev.set(`${key}_op`, state.op);
                    } else {
                        for (const v of state.values) prev.append(key, v);
                        // Persist a non-default operator even before any value is
                        // entered, so the operator selector doesn't snap back to
                        // the default ("is any of") on the next render.
                        if (state.op !== "in") prev.set(`${key}_op`, state.op);
                    }
                }
                return prev;
            });
        },
        [setSearchParams],
    );

    function setPreset(key: string | null) {
        setSearchParams((prev) => {
            if (key) {
                prev.set("preset", key);
                prev.delete("from");
                prev.delete("to");
            } else {
                prev.delete("preset");
            }
            return prev;
        });
    }

    function setOutcome(value: string | null) {
        setSearchParams((prev) => {
            if (value === "allow" || value === "deny") {
                prev.set("outcome", value);
            } else {
                prev.delete("outcome");
            }
            return prev;
        });
    }

    function setSort(column: SortColumn, dir: "asc" | "desc") {
        setSearchParams((prev) => {
            if (column === "created_at" && dir === "desc") {
                prev.delete("sort");
                prev.delete("order");
            } else {
                prev.set("sort", column);
                prev.set("order", dir);
            }
            return prev;
        });
    }

    // Build query params. Preset takes precedence over raw from/to.
    const presetMs = presetStr ? PRESET_MS[presetStr] : undefined;
    const query: Query = {
        outcome: outcomeStr === "allow" ? true : outcomeStr === "deny" ? false : undefined,
        from:
            presetMs !== undefined
                ? dayjs().subtract(presetMs, "millisecond").toISOString()
                : fromStr || undefined,
        to: presetMs !== undefined ? undefined : toStr || undefined,
        sort,
        order,
    };

    // Indexed writes onto the union-keyed query type collapse to an intersection
    // (`string[] & number[]`); a loose record view keeps each assignment honest.
    const q = query as Record<string, unknown>;
    for (const key of FILTER_COLUMN_KEYS) {
        const { op, values } = getColumnFilter(key);
        const isNullOp = op === "is_null" || op === "not_null";
        if (values.length === 0 && !isNullOp) continue;
        if (!isNullOp) q[key] = FILTER_COLUMNS[key].numeric ? values.map(Number) : values;
        if (op !== "in") q[`${key}_op`] = op;
    }

    const hasCustomTo = !!toStr && presetMs === undefined;
    const hasActiveFilters =
        !!(fromStr || toStr || outcomeStr) ||
        FILTER_COLUMN_KEYS.some((key) => isFilterActive(getColumnFilter(key)));

    function clearAll() {
        setSearchParams((prev) => {
            const next = new URLSearchParams();
            // Preserve the time-range preset and sort — they are view settings, not column filters
            for (const k of ["preset", "sort", "order"]) {
                if (prev.has(k)) next.set(k, prev.get(k)!);
            }
            return next;
        });
    }

    // Stable signature of all active filters + sort, used to reset pagination.
    // The cursor encodes the active sort, so sort changes must reset it too.
    const filterKey = JSON.stringify({
        preset: presetStr,
        from: fromStr,
        to: toStr,
        outcome: outcomeStr,
        sort,
        order,
        columns: FILTER_COLUMN_KEYS.map((key) => {
            const f = getColumnFilter(key);
            return [key, f.op, f.values];
        }),
    });

    return {
        queryParams: query,
        filterKey,
        presetStr,
        fromStr,
        toStr,
        outcomeStr,
        sort,
        order,
        hasCustomTo,
        hasActiveFilters,
        getColumnFilter,
        setColumnFilter,
        setPreset,
        setOutcome,
        setSort,
        setSearchParams,
        clearAll,
    };
}
