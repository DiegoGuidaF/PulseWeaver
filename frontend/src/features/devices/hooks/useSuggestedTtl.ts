import { useEffect, useState } from "react";
import { useSearchParams } from "react-router";
import { SUGGESTED_TTL_PARAM, parseSuggestedTtlSeconds } from "@/lib/ttlPresets";

/**
 * A lease TTL offered by a screen that measured one, read from `?suggest_ttl=`.
 * Latched at mount and then cleared from the URL, so the value survives the
 * strip but a reload does not re-offer one the user already dismissed. A value
 * the lease-rule contract would reject reads as no suggestion at all.
 */
export function useSuggestedTtl(): number | null {
    const [searchParams, setSearchParams] = useSearchParams();
    const [suggestedTtl] = useState(() =>
        parseSuggestedTtlSeconds(searchParams.get(SUGGESTED_TTL_PARAM)),
    );

    useEffect(() => {
        if (suggestedTtl === null) return;
        setSearchParams(
            (prev) => {
                prev.delete(SUGGESTED_TTL_PARAM);
                return prev;
            },
            { replace: true },
        );
    }, [suggestedTtl, setSearchParams]);

    return suggestedTtl;
}
