import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import dayjs from "dayjs";
import { AddressHistoryFilterOperator, type GetAddressHistoryHistogramData } from "@/lib/api";
import { DEFAULT_PRESET_KEY, PRESET_MS } from "@/lib/timePresets";
import {
    type ColumnFilterState,
    type FilterColumnKey,
    type FilterOp,
    FILTER_COLUMN_KEYS,
    FILTER_COLUMNS,
    isFilterActive,
} from "../filterConfig";

/** Filters + time window shared by both the events list and the histogram — the refetch boundary between them is the cursor, which lives outside this type. */
type Query = NonNullable<GetAddressHistoryHistogramData["query"]>;

/** Filter store key; bump the suffix when the default params change, so a stale entry can't restore a default the UI no longer offers. */
const LS_KEY = "pulseweaver:address-history:filters:v2";

/** A pre-applied, non-removable filter. The device tab locks `device_id`; a future user-scoped view locks `user_id` the same way. */
export interface LockedFilter {
    key: FilterColumnKey;
    values: string[];
}

export type SearchParamsSetter = (updater: URLSearchParams | ((prev: URLSearchParams) => URLSearchParams)) => void;

export interface AddressHistoryFilters {
    queryParams: Query;
    filterKey: string;

    presetStr: string | null;
    fromStr: string | null;
    toStr: string | null;

    hasCustomTo: boolean;
    hasActiveFilters: boolean;
    lockedFilter?: LockedFilter;

    getColumnFilter: (key: FilterColumnKey) => ColumnFilterState;
    setColumnFilter: (key: FilterColumnKey, state: ColumnFilterState | null) => void;

    setPreset: (key: string | null) => void;
    setSearchParams: SearchParamsSetter;
    clearAll: () => void;
}

/** Default params: the default time preset and nothing else — every column filter starts empty. */
export function buildDefaultParams(): URLSearchParams {
    return new URLSearchParams({ preset: DEFAULT_PRESET_KEY });
}

function persistFilters(params: URLSearchParams) {
    const str = params.toString();
    if (str) localStorage.setItem(LS_KEY, str);
    else localStorage.removeItem(LS_KEY);
}

interface FilterCoreOptions {
    locked?: LockedFilter;
}

// ─── Shared core ────────────────────────────────────────────────────────────

/**
 * Core filter logic shared between URL-backed and local-state-backed hooks.
 * Both hooks provide a URLSearchParams + setter pair; this hook maps filterx's
 * repeated-param + `{field}_op` model onto typed column filter state.
 */
export function useFilterCore(
    searchParams: URLSearchParams,
    setSearchParams: SearchParamsSetter,
    options?: FilterCoreOptions,
): AddressHistoryFilters {
    const locked = options?.locked;

    const presetStr = searchParams.get("preset");
    const fromStr = searchParams.get("from");
    const toStr = searchParams.get("to");

    const getColumnFilter = useCallback(
        (key: FilterColumnKey): ColumnFilterState => {
            if (locked && key === locked.key) {
                return { op: AddressHistoryFilterOperator.IN, values: locked.values };
            }
            return {
                op: (searchParams.get(`${key}_op`) as FilterOp | null) ?? AddressHistoryFilterOperator.IN,
                values: searchParams.getAll(key),
            };
        },
        [searchParams, locked],
    );

    const setColumnFilter = useCallback(
        (key: FilterColumnKey, state: ColumnFilterState | null) => {
            if (locked && key === locked.key) return;
            setSearchParams((prev) => {
                prev.delete(key);
                prev.delete(`${key}_op`);
                if (state) {
                    for (const v of state.values) prev.append(key, v);
                    // Persist a non-default operator even before any value is
                    // entered, so the operator selector doesn't snap back to
                    // the default ("is any of") on the next render.
                    if (state.op !== AddressHistoryFilterOperator.IN) prev.set(`${key}_op`, state.op);
                }
                return prev;
            });
        },
        [setSearchParams, locked],
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

    // Build query params. Preset takes precedence over raw from/to.
    const presetMs = presetStr ? PRESET_MS[presetStr] : undefined;
    const query: Query = {
        from: presetMs !== undefined
            ? dayjs().subtract(presetMs, "millisecond").toISOString()
            : (fromStr || undefined),
        to: presetMs !== undefined ? undefined : (toStr || undefined),
    };

    // Indexed writes onto the union-keyed query type collapse to an intersection
    // (`string[] & number[]`); a loose record view keeps each assignment honest.
    const q = query as Record<string, unknown>;
    for (const key of FILTER_COLUMN_KEYS) {
        const { op, values } = getColumnFilter(key);
        if (values.length === 0) continue;
        q[key] = FILTER_COLUMNS[key].numeric ? values.map(Number) : values;
        if (op !== AddressHistoryFilterOperator.IN) q[`${key}_op`] = op;
    }

    const hasCustomTo = !!toStr && presetMs === undefined;
    const hasActiveFilters =
        !!(fromStr || toStr) ||
        FILTER_COLUMN_KEYS.some((key) => {
            if (locked && key === locked.key) return false;
            return isFilterActive(getColumnFilter(key));
        });

    function clearAll() {
        setSearchParams((prev) => {
            const next = new URLSearchParams();
            // Preserve the time-range preset — it is a view setting, not a column filter.
            if (prev.has("preset")) next.set("preset", prev.get("preset")!);
            return next;
        });
    }

    // Stable signature of all active filters, used to reset pagination.
    const filterKey = JSON.stringify({
        preset: presetStr,
        from: fromStr,
        to: toStr,
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
        hasCustomTo,
        hasActiveFilters,
        lockedFilter: locked,
        getColumnFilter,
        setColumnFilter,
        setPreset,
        setSearchParams,
        clearAll,
    };
}

// ─── URL-backed hook (for dedicated page) ───────────────────────────────────

function getDefaultParams(): URLSearchParams {
    const saved = localStorage.getItem(LS_KEY);
    if (saved) return new URLSearchParams(saved);
    return buildDefaultParams();
}

/**
 * Computes the default URLSearchParams for the initial useSearchParams call.
 * Returns defaults from localStorage (or the global defaults) only when the
 * URL has no time context. When the URL already has `preset` or `from`,
 * returns undefined so useSearchParams uses the URL as-is.
 */
function computeInitialDefault(): URLSearchParams | undefined {
    if (typeof window === "undefined") return getDefaultParams();
    const url = new URLSearchParams(window.location.search);
    if (url.has("preset") || url.has("from")) return undefined;
    return getDefaultParams();
}

export function useAddressHistoryFilters(): AddressHistoryFilters {
    const [defaultInit] = useState(computeInitialDefault);
    const [searchParams, setSearchParamsRaw] = useSearchParams(defaultInit);

    // react-router branches each functional update off the same render-time
    // snapshot (`nextInit(new URLSearchParams(searchParams))`), so two updates
    // fired in one tick both start from the original params and the last write
    // wins — silently dropping the first. Closing the Device column's filter
    // popover does exactly this: it unmounts two ColumnFilters (device_id,
    // user_id) that each commit separately. Chaining off a ref that tracks the
    // latest emitted params lets such updates compose. The ref re-syncs to
    // searchParams after each commit.
    const latestParamsRef = useRef(searchParams);
    useEffect(() => {
        latestParamsRef.current = searchParams;
    });

    const setSearchParams: SearchParamsSetter = useCallback(
        (updater) => {
            setSearchParamsRaw(() => {
                const base = new URLSearchParams(latestParamsRef.current);
                const next = typeof updater === "function" ? updater(base) : updater;
                latestParamsRef.current = next;
                persistFilters(next);
                return next;
            });
        },
        [setSearchParamsRaw],
    );

    return useFilterCore(searchParams, setSearchParams);
}
