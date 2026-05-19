import type { GetDashboardTrafficData } from "@/lib/api";

type DashboardGranularity = NonNullable<NonNullable<GetDashboardTrafficData["query"]>["granularity"]>;

const MIN_5_MS = 5 * 60 * 1000;
const HOUR_MS = 60 * 60 * 1000;
const WEEK_MS = 7 * 24 * 60 * 60 * 1000;

export function granularityForRange(timeRangeMs: number): DashboardGranularity {
    if (timeRangeMs <= MIN_5_MS) return "minute";
    if (timeRangeMs <= HOUR_MS) return "5min";
    if (timeRangeMs <= WEEK_MS) return "hour";
    return "day";
}
