import { useState } from "react";
import { Stack, Title, Text, Group } from "@mantine/core";
import { AutoRefreshSelect } from "@/components/AutoRefreshSelect";
import { TimeRangePresetSelect } from "@/components/TimeRangePresetSelect";
import { AddressHistoryTable } from "@/features/address-history/components/AddressHistoryTable";
import { useAddressHistoryFilters } from "@/features/address-history/hooks/useAddressHistoryFilters";

const DEFAULT_REFRESH = 5_000;

export function AddressHistoryPage() {
    const filters = useAddressHistoryFilters();

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
                    <Title order={1}>Address Log</Title>
                    <Text c="dimmed">IP address lease events across all devices.</Text>
                </div>
                <Group gap="md">
                    <TimeRangePresetSelect value={filters.presetStr} onChange={filters.setPreset} />
                    <AutoRefreshSelect
                        value={refresh.interval}
                        onChange={(interval) => setRefresh((prev) => ({ ...prev, interval }))}
                    />
                </Group>
            </Group>
            <AddressHistoryTable filters={filters} refreshInterval={refresh.interval} />
        </Stack>
    );
}
