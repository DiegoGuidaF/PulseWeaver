import { useState } from "react";
import { Stack } from "@mantine/core";
import { AutoRefreshSelect, DEFAULT_REFRESH_INTERVAL } from "@/components/AutoRefreshSelect";
import { TimeRangePresetSelect } from "@/components/TimeRangePresetSelect";
import { PageToolbar } from "@/components/PageToolbar";
import { AccessLogTable } from "@/features/access-log/components/AccessLogTable";
import { useAccessLogFilters } from "@/features/access-log/hooks/useAccessLogFilters";

export function AccessLogPage() {
    const filters = useAccessLogFilters();

    const [refresh, setRefresh] = useState({
        hasCustomTo: filters.hasCustomTo,
        interval: filters.hasCustomTo ? 0 : DEFAULT_REFRESH_INTERVAL,
    });
    if (refresh.hasCustomTo !== filters.hasCustomTo) {
        setRefresh({
            hasCustomTo: filters.hasCustomTo,
            interval: filters.hasCustomTo ? 0 : DEFAULT_REFRESH_INTERVAL,
        });
    }

    return (
        <Stack gap="xl">
            <h1 style={{ position: "absolute", width: 1, height: 1, padding: 0, margin: -1, overflow: "hidden", clip: "rect(0,0,0,0)", whiteSpace: "nowrap", border: 0 }}>Access log</h1>
            <PageToolbar
                subtitle="Policy decisions"
                right={
                    <>
                        <TimeRangePresetSelect value={filters.presetStr} onChange={filters.setPreset} />
                        <AutoRefreshSelect
                            value={refresh.interval}
                            onChange={(interval) => setRefresh((prev) => ({ ...prev, interval }))}
                        />
                    </>
                }
            />
            <AccessLogTable filters={filters} refreshInterval={refresh.interval} />
        </Stack>
    );
}
