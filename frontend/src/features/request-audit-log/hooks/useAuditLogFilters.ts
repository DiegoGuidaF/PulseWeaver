import { useState, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { useDebouncedCallback } from "@mantine/hooks";
import dayjs from "dayjs";
import type { GetRequestAuditLogData } from "@/lib/api";
import { DEFAULT_PRESET_KEY, PRESET_MS } from "../constants";

const LS_KEY = "pulseweaver:audit-log:filters";
const DEFAULT_PARAMS = new URLSearchParams({ preset: DEFAULT_PRESET_KEY });

/** Save current search params to localStorage for next visit. */
function persistFilters(params: URLSearchParams) {
    const str = params.toString();
    if (str) localStorage.setItem(LS_KEY, str);
    else localStorage.removeItem(LS_KEY);
}

export interface AuditLogFilters {
    queryParams: GetRequestAuditLogData["query"];
    filterKey: string;

    // Individual values for UI widgets
    presetStr: string | null;
    deviceIdStr: string | null;
    outcomeStr: string | null;
    denyReason: string | null;
    fromStr: string | null;
    toStr: string | null;
    ipLocal: string;
    ipDebounced: string;

    /** True when a custom `to` is set (historical view, not live tail). */
    hasCustomTo: boolean;
    /** True when any filter is active (preset, date, device, outcome, IP, deny reason). */
    hasActiveFilters: boolean;

    // Setters
    setPreset: (key: string | null) => void;
    setParam: (key: string, value: string | null) => void;
    setIpLocal: (value: string) => void;
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

export function useAuditLogFilters(): AuditLogFilters {
    const [searchParams, setSearchParamsRaw] = useSearchParams(getInitialParams());

    // Wrap setSearchParams to persist every change to localStorage
    const setSearchParams: AuditLogFilters["setSearchParams"] = useCallback(
        (updater) => {
            setSearchParamsRaw((prev) => {
                const next = typeof updater === "function" ? updater(prev) : updater;
                persistFilters(next);
                return next;
            });
        },
        [setSearchParamsRaw],
    );

    // URL-derived filter values
    const presetStr = searchParams.get("preset");
    const deviceIdStr = searchParams.get("device_id");
    const outcomeStr = searchParams.get("outcome");
    const denyReason = searchParams.get("deny_reason") ?? null;
    const fromStr = searchParams.get("from");
    const toStr = searchParams.get("to");

    // IP filter: local state for responsive input, debounced write to URL.
    const [ipLocal, setIpLocalRaw] = useState(() => searchParams.get("ip") ?? "");
    const ipDebounced = searchParams.get("ip") ?? "";

    const syncIpToUrl = useDebouncedCallback((value: string) => {
        setSearchParams((prev) => {
            if (value === "") prev.delete("ip");
            else prev.set("ip", value);
            return prev;
        });
    }, 300);

    const setIpLocal = useCallback((value: string) => {
        setIpLocalRaw(value);
        syncIpToUrl(value);
    }, [syncIpToUrl]);

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
        setSearchParams((prev) => {
            if (value === null || value === "") prev.delete(key);
            else prev.set(key, value);
            return prev;
        });
    }

    // Compute query params: preset takes precedence over raw from/to
    const presetMs = presetStr ? PRESET_MS[presetStr] : undefined;
    const queryParams: GetRequestAuditLogData["query"] = {
        device_id: deviceIdStr ? Number(deviceIdStr) : undefined,
        outcome: outcomeStr === "allow" ? true : outcomeStr === "deny" ? false : undefined,
        ip: ipDebounced || undefined,
        deny_reason: denyReason || undefined,
        from: presetMs !== undefined
            ? dayjs().subtract(presetMs, "millisecond").toISOString()
            : (fromStr || undefined),
        to: presetMs !== undefined ? undefined : (toStr || undefined),
    };

    const hasCustomTo = !!toStr && presetMs === undefined;
    const hasNonDefaultPreset = !!presetStr && presetStr !== DEFAULT_PRESET_KEY;
    const hasActiveFilters = !!(hasNonDefaultPreset || fromStr || toStr || deviceIdStr || outcomeStr || denyReason || ipDebounced);

    function clearAll() {
        setIpLocalRaw("");
        syncIpToUrl.cancel();
        setSearchParams(new URLSearchParams(DEFAULT_PARAMS));
    }

    // Changes whenever any filter value changes — used to reset pagination.
    const filterKey = `${presetStr}|${deviceIdStr}|${outcomeStr}|${denyReason}|${fromStr}|${toStr}|${ipDebounced}`;

    return {
        queryParams,
        filterKey,
        presetStr,
        deviceIdStr,
        outcomeStr,
        denyReason,
        fromStr,
        toStr,
        ipLocal,
        ipDebounced,
        hasCustomTo,
        hasActiveFilters,
        setPreset,
        setParam,
        setIpLocal,
        setSearchParams,
        clearAll,
    };
}
