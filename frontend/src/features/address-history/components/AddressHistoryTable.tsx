import { useState } from "react";
import { Alert, Button, Group, Skeleton, Stack, Text } from "@mantine/core";
import { DataTable } from "mantine-datatable";
import { IconAlertCircle, IconFilterOff } from "@tabler/icons-react";
import type { AddressHistoryEvent } from "@/lib/api";
import { useAddressHistory } from "@/features/devices/hooks/useAddressHistory";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";
import { getAddressHistoryColumns } from "./addressHistoryColumns";
import { toErrorMessage } from "@/lib/api-client";
import { useDateFormatter, usePickerValueFormat } from "@/contexts/useDateTimePrefs";
import { useDevices } from "@/features/devices/hooks/useDevices";

interface AddressHistoryTableProps {
    filters: AddressHistoryFilters;
    refreshInterval: number;
}

export function AddressHistoryTable({ filters, refreshInterval }: AddressHistoryTableProps) {
    const formatDateTime = useDateFormatter();
    const pickerValueFormat = usePickerValueFormat();

    const [pagination, setPagination] = useState({
        filterKey: filters.filterKey,
        beforeId: undefined as number | undefined,
        allRows: [] as AddressHistoryEvent[],
    });
    if (pagination.filterKey !== filters.filterKey) {
        setPagination({ filterKey: filters.filterKey, beforeId: undefined, allRows: [] });
    }
    const { beforeId, allRows } = pagination;

    const { data: devices } = useDevices();

    const { data, isPending, error } = useAddressHistory(
        { ...filters.queryParams, before_id: beforeId },
        refreshInterval === 0 ? false : refreshInterval,
    );

    const events = data?.events ?? [];
    const rows = beforeId !== undefined
        ? allRows.concat(events)
        : events;

    function handleLoadMore() {
        if (!data?.next_cursor) return;
        setPagination((prev) => ({
            ...prev,
            allRows: rows,
            beforeId: data.next_cursor ?? undefined,
        }));
    }

    const deviceOptions = (devices ?? []).map((d) => ({ value: String(d.id), label: d.name }));

    const columns = getAddressHistoryColumns({
        formatDateTime,
        pickerValueFormat,
        presetStr: filters.presetStr,
        fromStr: filters.fromStr,
        toStr: filters.toStr,
        ipLocal: filters.ipLocal,
        ipDebounced: filters.ipDebounced,
        deviceIdStr: filters.deviceIdStr,
        sourceStr: filters.sourceStr,
        enabledStr: filters.enabledStr,
        deviceOptions,
        setParam: filters.setParam,
        setIpLocal: filters.setIpLocal,
        setSearchParams: filters.setSearchParams,
    });

    if ((isPending || !data) && !error && rows.length === 0) {
        return (
            <Stack gap="xs">
                {Array.from({ length: 5 }).map((_, i) => (
                    <Skeleton key={i} height={40} radius="sm" />
                ))}
            </Stack>
        );
    }

    if (error) {
        return (
            <Alert icon={<IconAlertCircle size={16} />} color="red" title="Failed to load">
                {toErrorMessage(error)}
            </Alert>
        );
    }

    const total = data?.total_events ?? 0;
    const hasMore = Boolean(data?.next_cursor);

    return (
        <Stack gap="sm">
            <Group justify="space-between">
                <Text size="sm" c="dimmed">
                    {total} event{total !== 1 ? "s" : ""}
                </Text>
                {filters.hasActiveFilters && (
                    <Button
                        variant="subtle"
                        size="compact-xs"
                        leftSection={<IconFilterOff size={14} />}
                        onClick={filters.clearAll}
                    >
                        Clear filters
                    </Button>
                )}
            </Group>

            <DataTable
                records={rows}
                idAccessor="id"
                highlightOnHover
                minHeight={150}
                noRecordsText="No address events found."
                columns={columns}
            />

            {hasMore && (
                <Group justify="center">
                    <Button
                        variant="subtle"
                        onClick={handleLoadMore}
                        loading={isPending}
                    >
                        Load more
                    </Button>
                </Group>
            )}
        </Stack>
    );
}
