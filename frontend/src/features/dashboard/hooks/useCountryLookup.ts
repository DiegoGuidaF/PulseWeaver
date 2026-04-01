import { useMemo } from "react";
import type { AuditLogCountryStats } from "@/lib/api/types.gen";

export function useCountryLookup(data: AuditLogCountryStats[] | undefined) {
    return useMemo(() => {
        if (!data) return new Map<string, AuditLogCountryStats>();
        return new Map(data.map((s) => [s.country_code, s]));
    }, [data]);
}
