import { useState } from "react";
import { Group, Stack } from "@mantine/core";
import { AutoRefreshSelect } from "@/components/AutoRefreshSelect";
import { TimeRangePresetSelect } from "@/components/TimeRangePresetSelect";
import { AddressHistoryTable } from "./AddressHistoryTable";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";

const DEFAULT_REFRESH = 5_000;

interface AddressHistoryViewProps {
    filters: AddressHistoryFilters;
}

export function AddressHistoryView({ filters }: AddressHistoryViewProps) {
    const [userInterval, setUserInterval] = useState(DEFAULT_REFRESH);
    const effectiveInterval = filters.hasCustomTo ? 0 : userInterval;

    return (
        <Stack gap="md">
            <Group gap="md" justify="flex-end">
                <TimeRangePresetSelect value={filters.presetStr} onChange={filters.setPreset} />
                <AutoRefreshSelect value={effectiveInterval} onChange={setUserInterval} />
            </Group>
            <AddressHistoryTable filters={filters} refreshInterval={effectiveInterval} />
        </Stack>
    );
}
