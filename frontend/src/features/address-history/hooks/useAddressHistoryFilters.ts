import { useState, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { useDebouncedCallback } from "@mantine/hooks";
import dayjs from "dayjs";
import type { GetAddressHistoryData } from "@/lib/api";
import { PRESET_MS } from "@/lib/timePresets";

const LS_KEY = "pulseweaver:address-history:filters";
const DEFAULT_PRESET = "last_24h";
const DEFAULT_PARAMS = new URLSearchParams({ preset: DEFAULT_PRESET });

function persistFilters(params: URLSearchParams) {
    const str = params.toString();
    if (str) localStorage.setItem(LS_KEY, str);
    else localStorage.removeItem(LS_KEY);
}

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

    setPreset: (key: string | null) => void;
    setParam: (key: string, value: string | null) => void;
    setIpLocal: (value: string) => void;
    setSearchParams: (updater: URLSearchParams | ((prev: URLSearchParams) => URLSearchParams)) => void;
    clearAll: () => void;
}

function getDefaultParams(): URLSearchParams {
    const saved = localStorage.getItem(LS_KEY);
    if (saved) return new URLSearchParams(saved);
    return new URLSearchParams(DEFAULT_PARAMS);
}

export function useAddressHistoryFilters(): AddressHistoryFilters {
    // useSearchParams uses the router's location (works with MemoryRouter in tests).
    // The default is applied only when the URL has no search params.
    const [searchParams, setSearchParamsRaw] = useSearchParams(getDefaultParams());

    const setSearchParams: AddressHistoryFilters["setSearchParams"] = useCallback(
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
    const deviceIdStr = searchParams.get("device_id");
    const sourceStr = searchParams.get("source");
    const enabledStr = searchParams.get("is_enabled");
    const fromStr = searchParams.get("from");
    const toStr = searchParams.get("to");

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

    const presetMs = presetStr ? PRESET_MS[presetStr] : undefined;
    const queryParams: GetAddressHistoryData["query"] = {
        device_id: deviceIdStr ? [Number(deviceIdStr)] : undefined,
        source: (sourceStr || undefined) as "heartbeat" | "manual" | "expiry" | undefined,
        is_enabled: enabledStr === "true" ? true : enabledStr === "false" ? false : undefined,
        ip: ipDebounced || undefined,
        from: presetMs !== undefined
            ? dayjs().subtract(presetMs, "millisecond").toISOString()
            : (fromStr || undefined),
        to: presetMs !== undefined ? undefined : (toStr || undefined),
    };

    const hasCustomTo = !!toStr && presetMs === undefined;
    const hasNonDefaultPreset = !!presetStr && presetStr !== DEFAULT_PRESET;
    const hasActiveFilters = !!(hasNonDefaultPreset || fromStr || toStr || deviceIdStr || sourceStr || enabledStr || ipDebounced);

    function clearAll() {
        setIpLocalRaw("");
        syncIpToUrl.cancel();
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
        setPreset,
        setParam,
        setIpLocal,
        setSearchParams,
        clearAll,
    };
}
