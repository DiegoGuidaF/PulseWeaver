import { useState, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { useDebouncedCallback } from "@mantine/hooks";
import dayjs from "dayjs";
import type { GetAddressHistoryData } from "@/lib/api";
import { DEFAULT_PRESET_KEY, PRESET_MS } from "@/lib/timePresets";

const LS_KEY = "pulseweaver:address-history:filters";
const DEFAULT_PARAMS = new URLSearchParams({ preset: DEFAULT_PRESET_KEY });

export type SearchParamsSetter = (updater: URLSearchParams | ((prev: URLSearchParams) => URLSearchParams)) => void;

export interface AddressHistoryFilters {
    queryParams: GetAddressHistoryData["query"];
    filterKey: string;

    presetStr: string | null;
    deviceIdStr: string | null;
    sourceStr: string | null;
    enabledStr: string | null;
    fromStr: string | null;
    toStr: string | null;
    ipLocal: string;
    ipDebounced: string;

    hasCustomTo: boolean;
    hasActiveFilters: boolean;
    lockedDeviceId?: number;

    setPreset: (key: string | null) => void;
    setParam: (key: string, value: string | null) => void;
    setIpLocal: (value: string) => void;
    setSearchParams: SearchParamsSetter;
    clearAll: () => void;
}

function persistFilters(params: URLSearchParams) {
    const str = params.toString();
    if (str) localStorage.setItem(LS_KEY, str);
    else localStorage.removeItem(LS_KEY);
}

function getDefaultParams(): URLSearchParams {
    const saved = localStorage.getItem(LS_KEY);
    if (saved) return new URLSearchParams(saved);
    return new URLSearchParams(DEFAULT_PARAMS);
}

// ─── Shared core ────────────────────────────────────────────────────────────

interface FilterCoreOptions {
    lockedDeviceId?: number;
}

/**
 * Core filter logic shared between URL-backed and local-state-backed hooks.
 * Both hooks provide a URLSearchParams + setter pair; this hook handles
 * IP debounce, derived state, and query param computation.
 */
export function useFilterCore(
    searchParams: URLSearchParams,
    setSearchParams: SearchParamsSetter,
    options?: FilterCoreOptions,
): AddressHistoryFilters {
    const lockedDeviceId = options?.lockedDeviceId;

    const presetStr = searchParams.get("preset");
    const deviceIdStr = lockedDeviceId != null ? String(lockedDeviceId) : searchParams.get("device_id");
    const sourceStr = searchParams.get("source");
    const enabledStr = searchParams.get("is_enabled");
    const fromStr = searchParams.get("from");
    const toStr = searchParams.get("to");

    const [ipLocal, setIpLocalRaw] = useState(() => searchParams.get("ip") ?? "");
    const ipDebounced = searchParams.get("ip") ?? "";

    const syncIpToParams = useDebouncedCallback((value: string) => {
        setSearchParams((prev) => {
            if (value === "") prev.delete("ip");
            else prev.set("ip", value);
            return prev;
        });
    }, 300);

    const setIpLocal = useCallback((value: string) => {
        setIpLocalRaw(value);
        syncIpToParams(value);
    }, [syncIpToParams]);

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

    function setParam(key: string, value: string | null) {
        if (key === "device_id" && lockedDeviceId != null) return;
        setSearchParams((prev) => {
            if (value === null || value === "") prev.delete(key);
            else prev.set(key, value);
            return prev;
        });
    }

    const presetMs = presetStr ? PRESET_MS[presetStr] : undefined;
    const queryParams: GetAddressHistoryData["query"] = {
        device_id: lockedDeviceId != null
            ? [lockedDeviceId]
            : deviceIdStr ? [Number(deviceIdStr)] : undefined,
        source: (sourceStr || undefined) as "heartbeat" | "manual" | "expiry" | undefined,
        is_enabled: enabledStr === "true" ? true : enabledStr === "false" ? false : undefined,
        ip: ipDebounced || undefined,
        from: presetMs !== undefined
            ? dayjs().subtract(presetMs, "millisecond").toISOString()
            : (fromStr || undefined),
        to: presetMs !== undefined ? undefined : (toStr || undefined),
    };

    const hasCustomTo = !!toStr && presetMs === undefined;
    const hasNonDefaultPreset = !!presetStr && presetStr !== DEFAULT_PRESET_KEY;
    const hasActiveFilters = !!(hasNonDefaultPreset || fromStr || toStr || deviceIdStr || sourceStr || enabledStr || ipDebounced);

    function clearAll() {
        setIpLocalRaw("");
        syncIpToParams.cancel();
        setSearchParams(new URLSearchParams(DEFAULT_PARAMS));
    }

    const filterKey = `${presetStr}|${deviceIdStr}|${sourceStr}|${enabledStr}|${fromStr}|${toStr}|${ipDebounced}`;

    return {
        queryParams,
        filterKey,
        presetStr,
        deviceIdStr,
        sourceStr,
        enabledStr,
        fromStr,
        toStr,
        ipLocal,
        ipDebounced,
        hasCustomTo,
        hasActiveFilters,
        lockedDeviceId,
        setPreset,
        setParam,
        setIpLocal,
        setSearchParams,
        clearAll,
    };
}

// ─── URL-backed hook (for dedicated page) ───────────────────────────────────

/**
 * Computes the default URLSearchParams for the initial useSearchParams call.
 * Returns defaults from localStorage (or the global default preset) only when
 * the URL has no time context. When the URL already has `preset` or `from`,
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

    const setSearchParams: SearchParamsSetter = useCallback(
        (updater) => {
            setSearchParamsRaw((prev) => {
                const next = typeof updater === "function" ? updater(prev) : updater;
                persistFilters(next);
                return next;
            });
        },
        [setSearchParamsRaw],
    );

    return useFilterCore(searchParams, setSearchParams);
}
