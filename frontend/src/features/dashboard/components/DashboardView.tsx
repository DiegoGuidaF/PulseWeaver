import { useMemo } from "react";
import dayjs from "dayjs";
import { SimpleGrid, Stack, Group, Text } from "@mantine/core";
import { useDashboardPosture } from "../hooks/useDashboardPosture";
import { useDashboardStats } from "../hooks/useDashboardStats";
import { useDashboardTraffic } from "../hooks/useDashboardTraffic";
import { useDashboardServices } from "../hooks/useDashboardServices";
import { useTopDeniedIPs } from "../hooks/useTopDeniedIPs";
import { useDashboardTimeRange } from "../hooks/useDashboardTimeRange";
import { PostureStrip } from "./PostureStrip";
import { DashboardStatCards } from "./DashboardStatCards";
import { TrafficLineChart } from "@/components/TrafficLineChart";
import { ServiceBarChart } from "./ServiceBarChart";
import { TopDeniedIPsTable } from "./TopDeniedIPsTable";
import { CountryStatsSection } from "./CountryStatsSection";
import { AttributionSection } from "./AttributionSection";
import { TimeRangePresetSelect } from "@/components/TimeRangePresetSelect";
import { DEFAULT_PRESET_KEY, PRESET_MS, type PresetKey } from "@/lib/timePresets";
import { useAddressHistoryTuning } from "@/features/address-history/hooks/useAddressHistoryTuning";

/**
 * Lookback for the TTL-tuning signal. Posture is otherwise a "right now"
 * reading, so this one is windowed — the same key drives both the count and the
 * link out, which is what lets the destination reproduce the number.
 */
const TUNING_PRESET: PresetKey = "last_1w";

export function DashboardView() {
    const { from, to, presetKey, setPresetKey } = useDashboardTimeRange();
    const timeRangeMs = PRESET_MS[presetKey] ?? PRESET_MS[DEFAULT_PRESET_KEY];

    const posture = useDashboardPosture();
    // Resolved once per mount: a window recomputed each render would give React
    // Query a key that never repeats.
    const tuningWindow = useMemo(
        () => ({ from: dayjs().subtract(PRESET_MS[TUNING_PRESET], "millisecond").toISOString() }),
        [],
    );
    const tuning = useAddressHistoryTuning(tuningWindow);
    const stats = useDashboardStats(from, to);
    const traffic = useDashboardTraffic(from, to);
    const services = useDashboardServices(from, to);
    const topDenied = useTopDeniedIPs(from, to);

    return (
        <Stack gap="xl">
            <PostureStrip
                data={posture.data}
                isLoading={posture.isLoading}
                error={posture.error}
                onRetry={() => posture.refetch()}
                tuningCount={tuning.data?.total ?? 0}
                tuningPreset={TUNING_PRESET}
            />

            <Stack gap="lg">
                {/* Preset lives on the traffic section, not the page toolbar, so it is
                    unambiguous that it scopes traffic only — posture above is "now". */}
                <Group justify="space-between" align="center" wrap="wrap">
                    <Text fw={600}>Traffic</Text>
                    <TimeRangePresetSelect
                        value={presetKey}
                        onChange={(key) => setPresetKey(key ?? DEFAULT_PRESET_KEY)}
                    />
                </Group>

                <DashboardStatCards
                    data={stats.data}
                    isLoading={stats.isLoading}
                    error={stats.error}
                    onRetry={() => stats.refetch()}
                />

                <AttributionSection from={from} to={to} />

                <SimpleGrid cols={{ base: 1, md: 2 }}>
                    <TrafficLineChart
                        data={traffic.data?.buckets}
                        isLoading={traffic.isLoading}
                        timeRangeMs={timeRangeMs}
                        error={traffic.error}
                        onRetry={() => traffic.refetch()}
                    />
                    <ServiceBarChart
                        data={services.data?.services}
                        isLoading={services.isLoading}
                        error={services.error}
                        onRetry={() => services.refetch()}
                    />
                </SimpleGrid>

                <CountryStatsSection from={from} to={to} />

                <TopDeniedIPsTable
                    data={topDenied.data?.ips}
                    isLoading={topDenied.isLoading}
                    error={topDenied.error}
                    onRetry={() => topDenied.refetch()}
                />
            </Stack>
        </Stack>
    );
}
