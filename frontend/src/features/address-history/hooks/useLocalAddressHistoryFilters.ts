import { useState, useCallback } from "react";
import type { AddressHistoryFilters, SearchParamsSetter } from "./useAddressHistoryFilters";
import { useFilterCore } from "./useAddressHistoryFilters";

const DEFAULT_PARAMS = new URLSearchParams({ preset: "last_24h" });

interface UseLocalAddressHistoryFiltersOptions {
    lockedDeviceId: number;
}

/**
 * Local-state-backed address history filters for embedded use (e.g. device detail tab).
 * Uses useState instead of URL search params — no URL pollution, no localStorage persistence.
 */
export function useLocalAddressHistoryFilters(
    options: UseLocalAddressHistoryFiltersOptions,
): AddressHistoryFilters {
    const [params, setParamsRaw] = useState(() => new URLSearchParams(DEFAULT_PARAMS));

    const setSearchParams: SearchParamsSetter = useCallback((updater) => {
        setParamsRaw((prev) => {
            const next = typeof updater === "function" ? updater(new URLSearchParams(prev)) : updater;
            return next;
        });
    }, []);

    return useFilterCore(params, setSearchParams, { lockedDeviceId: options.lockedDeviceId });
}
