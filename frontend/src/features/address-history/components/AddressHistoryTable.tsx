import { useState, useMemo } from "react";
import { Alert, Button, Card, Group, Skeleton, Stack, Text } from "@mantine/core";
import { LineChart } from "@mantine/charts";
import { DataTable } from "mantine-datatable";
import { IconAlertCircle, IconFilterOff } from "@tabler/icons-react";
import dayjs from "dayjs";
import type { AddressHistoryEvent } from "@/lib/api";
import { ActiveFilterChips, type FilterChip } from "@/components/ActiveFilterChips";
import { useAddressHistory } from "../hooks/useAddressHistory";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";
import { getAddressHistoryColumns } from "./addressHistoryColumns";
import { SOURCE_LABELS } from "../constants";
import { toErrorMessage } from "@/lib/api-client";
import { useDateFormatter, usePickerValueFormat } from "@/contexts/useDateTimePrefs";
import { useDevices } from "@/features/devices/hooks/useDevices";
import { PRESET_MS } from "@/lib/timePresets";

const THREE_DAYS_MS = 3 * 24 * 60 * 60 * 1000;

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
        lockedDeviceId: filters.lockedDeviceId,
        deviceOptions,
        setParam: filters.setParam,
        setIpLocal: filters.setIpLocal,
        setSearchParams: filters.setSearchParams,
    });

    const filterChips = useMemo(() => {
        const chips: FilterChip[] = [];

        if (filters.fromStr || filters.toStr) {
            const from = filters.fromStr ? formatDateTime(filters.fromStr) : "—";
            const to = filters.toStr ? formatDateTime(filters.toStr) : "now";
            chips.push({
                label: "Time",
                value: `${from} → ${to}`,
                onRemove: () => {
                    filters.setSearchParams((prev) => {
                        prev.delete("from");
                        prev.delete("to");
                        return prev;
                    });
                },
            });
        }

        if (filters.deviceIdStr && filters.lockedDeviceId == null) {
            const device = deviceOptions.find((d) => d.value === filters.deviceIdStr);
            chips.push({
                label: "Device",
                value: device?.label ?? filters.deviceIdStr,
                onRemove: () => filters.setParam("device_id", null),
            });
        }

        if (filters.ipDebounced) {
            chips.push({
                label: "IP",
                value: filters.ipDebounced,
                onRemove: () => filters.setIpLocal(""),
            });
        }

        if (filters.enabledStr) {
            chips.push({
                label: "Status",
                value: filters.enabledStr === "true" ? "Enabled" : "Disabled",
                onRemove: () => filters.setParam("is_enabled", null),
            });
        }

        if (filters.sourceStr) {
            chips.push({
                label: "Source",
                value: SOURCE_LABELS[filters.sourceStr] ?? filters.sourceStr,
                onRemove: () => filters.setParam("source", null),
            });
        }

        return chips;
    }, [filters, formatDateTime, deviceOptions]);

    // Chart data from buckets
    const useDayFormat = useMemo(() => {
        const presetMs = filters.presetStr ? PRESET_MS[filters.presetStr] : undefined;
        if (presetMs !== undefined) return presetMs >= THREE_DAYS_MS;
        if (filters.fromStr && filters.toStr) {
            return dayjs(filters.toStr).diff(dayjs(filters.fromStr)) >= THREE_DAYS_MS;
        }
        return false;
    }, [filters.presetStr, filters.fromStr, filters.toStr]);

    const chartData = useMemo(() => {
        if (!data?.buckets) return [];
        return data.buckets.map((b) => ({
            timestamp: dayjs(b.timestamp).format(useDayFormat ? "MMM DD" : "MMM DD HH:mm"),
            active_count: b.active_count,
        }));
    }, [data, useDayFormat]);

    if ((isPending || !data) && !error && rows.length === 0) {
        return (
            <Stack gap="md">
                <Skeleton height={200} radius="sm" />
                <Stack gap="xs">
                    {Array.from({ length: 5 }).map((_, i) => (
                        <Skeleton key={i} height={40} radius="sm" />
                    ))}
                </Stack>
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
            <Card withBorder padding="md">
                <Text fw={500} mb="sm">Active IPs over time</Text>
                {chartData.length > 0 ? (
                    <LineChart
                        h={200}
                        data={chartData}
                        dataKey="timestamp"
                        series={[{ name: "active_count", color: "orange.4", label:"Distinct IPs count" }]}
                        yAxisLabel="Distinct IPs"
                        curveType="monotone"
                        tooltipAnimationDuration={150}
                    />
                ) : (
                    <Text size="sm" c="dimmed" ta="center" py="xl">
                        No activity in this period
                    </Text>
                )}
            </Card>

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

            <ActiveFilterChips chips={filterChips} />

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
