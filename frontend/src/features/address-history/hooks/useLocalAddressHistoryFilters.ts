import { useCallback, useState } from "react";
import type { AddressHistoryFilters, LockedFilter, SearchParamsSetter } from "./useAddressHistoryFilters";
import { buildDefaultParams, useFilterCore } from "./useAddressHistoryFilters";

interface UseLocalAddressHistoryFiltersOptions {
    locked: LockedFilter;
}

/**
 * Local-state-backed address history filters for embedded use (e.g. device detail tab).
 * Uses useState instead of URL search params — no URL pollution, no localStorage persistence.
 */
export function useLocalAddressHistoryFilters(
    options: UseLocalAddressHistoryFiltersOptions,
): AddressHistoryFilters {
    const [params, setParamsRaw] = useState(buildDefaultParams);

    const setSearchParams: SearchParamsSetter = useCallback((updater) => {
        setParamsRaw((prev) => {
            const next = typeof updater === "function" ? updater(new URLSearchParams(prev)) : updater;
            return next;
        });
    }, []);

    return useFilterCore(params, setSearchParams, { locked: options.locked });
}
