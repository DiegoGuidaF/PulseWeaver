import { useState } from "react";
import { Stack } from "@mantine/core";
import { AutoRefreshSelect } from "@/components/AutoRefreshSelect";
import { TimeRangePresetSelect } from "@/components/TimeRangePresetSelect";
import { PageToolbar } from "@/components/PageToolbar";
import { RequestAuditLogTable } from "@/features/request-audit-log/components/RequestAuditLogTable";
import { useAuditLogFilters } from "@/features/request-audit-log/hooks/useAuditLogFilters";

const DEFAULT_REFRESH = 5_000;

export function RequestAuditLogPage() {
    const filters = useAuditLogFilters();

    const [refresh, setRefresh] = useState({
        hasCustomTo: filters.hasCustomTo,
        interval: filters.hasCustomTo ? 0 : DEFAULT_REFRESH,
    });
    if (refresh.hasCustomTo !== filters.hasCustomTo) {
        setRefresh({
            hasCustomTo: filters.hasCustomTo,
            interval: filters.hasCustomTo ? 0 : DEFAULT_REFRESH,
        });
    }

    return (
        <Stack maw={1200} gap="xl">
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
            <RequestAuditLogTable filters={filters} refreshInterval={refresh.interval} />
        </Stack>
    );
}
