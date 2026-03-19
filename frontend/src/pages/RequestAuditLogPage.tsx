import { useState } from "react";
import { Stack, Title, Text, Group } from "@mantine/core";
import { AutoRefreshSelect } from "@/components/AutoRefreshSelect";
import { TimeRangePresetSelect } from "@/features/request-audit-log/components/TimeRangePresetSelect";
import { RequestAuditLogTable } from "@/features/request-audit-log/components/RequestAuditLogTable";
import { useAuditLogFilters } from "@/features/request-audit-log/hooks/useAuditLogFilters";

const DEFAULT_REFRESH = 5_000;

export function RequestAuditLogPage() {
    const filters = useAuditLogFilters();

    // Bundle hasCustomTo into state so we can reset refreshInterval during
    // render when it changes (same pattern as pagination reset in the table).
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
            <Group justify="space-between" align="flex-end">
                <div>
                    <Title order={1}>Access Log</Title>
                    <Text c="dimmed">Policy decision history for all incoming requests.</Text>
                </div>
                <Group gap="md">
                    <TimeRangePresetSelect value={filters.presetStr} onChange={filters.setPreset} />
                    <AutoRefreshSelect
                        value={refresh.interval}
                        onChange={(interval) => setRefresh((prev) => ({ ...prev, interval }))}
                    />
                </Group>
            </Group>
            <RequestAuditLogTable filters={filters} refreshInterval={refresh.interval} />
        </Stack>
    );
}
