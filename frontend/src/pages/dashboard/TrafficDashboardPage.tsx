import { Stack } from "@mantine/core";
import { TimeRangePresetSelect } from "@/components/TimeRangePresetSelect";
import { PageToolbar } from "@/components/PageToolbar";
import { DashboardView } from "@/features/dashboard/components/DashboardView";
import { useDashboardTimeRange } from "@/features/dashboard/hooks/useDashboardTimeRange";
import { DEFAULT_PRESET_KEY, PRESET_MS } from "@/lib/timePresets";

export function TrafficDashboardPage() {
    const { from, to, presetKey, setPresetKey } = useDashboardTimeRange();
    const timeRangeMs = PRESET_MS[presetKey] ?? PRESET_MS[DEFAULT_PRESET_KEY];

    return (
        <Stack gap="xl">
            <h1 style={{ position: "absolute", width: 1, height: 1, padding: 0, margin: -1, overflow: "hidden", clip: "rect(0,0,0,0)", whiteSpace: "nowrap", border: 0 }}>Traffic</h1>
            <PageToolbar
                subtitle="Traffic overview"
                right={
                    <TimeRangePresetSelect
                        value={presetKey}
                        onChange={(key) => setPresetKey(key ?? DEFAULT_PRESET_KEY)}
                    />
                }
            />
            <DashboardView from={from} to={to} timeRangeMs={timeRangeMs} />
        </Stack>
    );
}
