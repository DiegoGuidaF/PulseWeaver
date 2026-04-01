import { useMemo } from "react";
import { useMantineTheme } from "@mantine/core";
import { scaleSqrt } from "d3-scale";
import type { AccessLogCountryStats } from "@/lib/api/types.gen";

type Metric = "denied" | "total";

export function useMapColorScale(
    data: AccessLogCountryStats[] | undefined,
    metric: Metric,
): (countryCode: string, lookup: Map<string, AccessLogCountryStats>) => string {
    const theme = useMantineTheme();

    return useMemo(() => {
        const maxVal = Math.max(
            1,
            ...(data ?? []).map((s) =>
                metric === "denied" ? s.denied : s.total,
            ),
        );
        const palette =
            metric === "denied"
                ? [theme.colors.red[1], theme.colors.red[7]]
                : [theme.colors.indigo[1], theme.colors.indigo[6]];
        const scale = scaleSqrt<string>().domain([0, maxVal]).range(palette);
        const noDataColor = theme.colors.dark[5];
        const zeroColor = theme.colors.dark[4];

        return (
            countryCode: string,
            lookup: Map<string, AccessLogCountryStats>,
        ) => {
            const stats = lookup.get(countryCode);
            if (!stats) return noDataColor;
            const val = metric === "denied" ? stats.denied : stats.total;
            return val === 0 ? zeroColor : scale(val);
        };
    }, [data, metric, theme]);
}
