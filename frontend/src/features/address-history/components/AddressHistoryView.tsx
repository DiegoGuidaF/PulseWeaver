import { useState } from "react";
import { SegmentedControl, Stack } from "@mantine/core";
import { AutoRefreshSelect, DEFAULT_REFRESH_INTERVAL } from "@/components/AutoRefreshSelect";
import { TimeRangePresetSelect } from "@/components/TimeRangePresetSelect";
import { PageToolbar } from "@/components/PageToolbar";
import { AddressHistoryTable } from "./AddressHistoryTable";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";

interface AddressHistoryViewProps {
    filters: AddressHistoryFilters;
    subtitle?: string;
}

export function AddressHistoryView({ filters, subtitle }: AddressHistoryViewProps) {
    const [userInterval, setUserInterval] = useState(DEFAULT_REFRESH_INTERVAL);
    const effectiveInterval = filters.hasCustomTo ? 0 : userInterval;

    return (
        <Stack gap="md">
            <PageToolbar
                subtitle={subtitle}
                left={
                    <SegmentedControl
                        size="xs"
                        data={[
                            { label: "State changes", value: "changes" },
                            { label: "All events", value: "all" },
                        ]}
                        value={filters.includeAll ? "all" : "changes"}
                        onChange={(val) => filters.setIncludeAll(val === "all")}
                    />
                }
                right={
                    <>
                        <TimeRangePresetSelect value={filters.presetStr} onChange={filters.setPreset} />
                        <AutoRefreshSelect value={effectiveInterval} onChange={setUserInterval} />
                    </>
                }
            />
            <AddressHistoryTable filters={filters} refreshInterval={effectiveInterval} />
        </Stack>
    );
}
