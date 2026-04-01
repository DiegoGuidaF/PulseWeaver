import { useMemo } from "react";
import type { AccessLogCountryStats } from "@/lib/api/types.gen";

export function useCountryLookup(data: AccessLogCountryStats[] | undefined) {
    return useMemo(() => {
        if (!data) return new Map<string, AccessLogCountryStats>();
        return new Map(data.map((s) => [s.country_code, s]));
    }, [data]);
}
