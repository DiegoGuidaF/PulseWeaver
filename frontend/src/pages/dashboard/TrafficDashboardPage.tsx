import { Stack } from "@mantine/core";
import { TimeRangePresetSelect } from "@/components/TimeRangePresetSelect";
import { PageToolbar } from "@/components/PageToolbar";
import { DashboardView } from "@/features/dashboard/components/DashboardView";
import { useDashboardTimeRange } from "@/features/dashboard/hooks/useDashboardTimeRange";
import { DEFAULT_PRESET_KEY, PRESET_MS } from "@/lib/timePresets";
import { granularityForRange } from "@/lib/granularity";

export function TrafficDashboardPage() {
    const { from, to, presetKey, setPresetKey } = useDashboardTimeRange();
    const timeRangeMs = PRESET_MS[presetKey] ?? PRESET_MS[DEFAULT_PRESET_KEY];
    const granularity = granularityForRange(timeRangeMs);

    return (
        <Stack gap="xl">
            <PageToolbar
                subtitle="Traffic overview"
                right={
                    <TimeRangePresetSelect
                        value={presetKey}
                        onChange={(key) => setPresetKey(key ?? DEFAULT_PRESET_KEY)}
                    />
                }
            />
            <DashboardView from={from} to={to} timeRangeMs={timeRangeMs} granularity={granularity} />
        </Stack>
    );
}
