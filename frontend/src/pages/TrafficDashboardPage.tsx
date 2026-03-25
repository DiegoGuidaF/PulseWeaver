import { Stack, Title, Text, Group } from "@mantine/core";
import { TimeRangePresetSelect } from "@/components/TimeRangePresetSelect";
import { DashboardView } from "@/features/dashboard/components/DashboardView";
import { useDashboardTimeRange } from "@/features/dashboard/hooks/useDashboardTimeRange";
import { DEFAULT_PRESET_KEY } from "@/lib/timePresets";

export function TrafficDashboardPage() {
    const { from, to, presetKey, setPresetKey } = useDashboardTimeRange();

    return (
        <Stack gap="xl">
            <Group justify="space-between" align="flex-end">
                <div>
                    <Title order={1}>Dashboard</Title>
                    <Text c="dimmed">Traffic overview from aggregated policy decisions.</Text>
                </div>
                <TimeRangePresetSelect value={presetKey} onChange={(key) => setPresetKey(key ?? DEFAULT_PRESET_KEY)} />
            </Group>
            <DashboardView from={from} to={to} />
        </Stack>
    );
}
