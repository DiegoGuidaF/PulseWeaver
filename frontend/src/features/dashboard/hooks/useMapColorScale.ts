import { useMemo } from "react";
import { useMantineTheme } from "@mantine/core";
import { scaleSqrt } from "d3-scale";
import type { AccessLogCountryStats } from "@/lib/api/types.gen";

function denyRate(s: AccessLogCountryStats): number {
    return s.total > 0 ? s.denied / s.total : 0;
}

/**
 * Colors each country by its deny rate (denied / total) rather than raw volume:
 * the map answers "where is traffic getting blocked", which volume already shows
 * elsewhere on the dashboard. Intensity is normalized to the busiest-blocking
 * country so low overall deny rates still produce a readable gradient.
 */
export function useMapColorScale(
    data: AccessLogCountryStats[] | undefined,
): (countryCode: string, lookup: Map<string, AccessLogCountryStats>) => string {
    const theme = useMantineTheme();

    return useMemo(() => {
        const maxRate = Math.max(0, ...(data ?? []).map(denyRate));
        const scale = scaleSqrt<string>()
            .domain([0, maxRate || 1])
            .range([theme.colors.red[3], theme.colors.red[8]]);
        const noDataColor = theme.colors.dark[6];
        const cleanColor = theme.colors.dark[3];

        return (
            countryCode: string,
            lookup: Map<string, AccessLogCountryStats>,
        ) => {
            const stats = lookup.get(countryCode);
            if (!stats) return noDataColor;
            // Traffic but nothing denied reads as neutral, not faint red.
            if (stats.denied === 0) return cleanColor;
            return scale(denyRate(stats));
        };
    }, [data, theme]);
}
