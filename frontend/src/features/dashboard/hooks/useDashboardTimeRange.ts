import { useState, useMemo } from "react";
import dayjs from "dayjs";
import { DEFAULT_PRESET_KEY, PRESET_MS } from "@/lib/timePresets";

export interface DashboardTimeRange {
    from: string;
    to: string | undefined;
    presetKey: string;
    setPresetKey: (key: string) => void;
}

export function useDashboardTimeRange(): DashboardTimeRange {
    const [presetKey, setPresetKey] = useState<string>(DEFAULT_PRESET_KEY);

    const { from, to } = useMemo(() => {
        const ms = PRESET_MS[presetKey];
        if (ms) {
            return {
                from: dayjs().subtract(ms, "millisecond").toISOString(),
                to: undefined,
            };
        }
        return {
            from: dayjs().subtract(PRESET_MS[DEFAULT_PRESET_KEY], "millisecond").toISOString(),
            to: undefined,
        };
    }, [presetKey]);

    return { from, to, presetKey, setPresetKey };
}
