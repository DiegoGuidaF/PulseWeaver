import dayjs from "dayjs";
import { PRESET_MS } from "@/lib/timePresets";

const HOUR_MS = 60 * 60 * 1000;
const DAY_MS = 24 * HOUR_MS;
const WEEK_MS = 7 * DAY_MS;

/**
 * Shared chart x-axis date formatter.
 * Adapts the label format based on the active time range.
 *
 * @param timestamp - ISO 8601 timestamp string
 * @param timeRangeMs - Active time range in milliseconds (from PRESET_MS or computed)
 */
export function formatChartLabel(timestamp: string, timeRangeMs: number): string {
    const d = dayjs(timestamp);

    if (timeRangeMs <= DAY_MS) {
        return d.format("HH:mm");
    }
    if (timeRangeMs <= WEEK_MS) {
        return d.format("MMM D HH:mm");
    }
    return d.format("MMM D");
}

/** Resolves a preset key (e.g. "last_1mo") to its millisecond value. */
export function presetToMs(presetKey: string): number {
    return PRESET_MS[presetKey] ?? DAY_MS;
}
