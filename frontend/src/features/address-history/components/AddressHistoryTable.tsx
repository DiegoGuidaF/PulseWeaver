import { useState, useMemo } from "react";
import { buildRoute } from "@/lib/routes";
import { useNavigate } from "react-router-dom";
import { ActionIcon, Button, Card, Group, Paper, Skeleton, Stack, Text, Tooltip } from "@mantine/core";
import { LineChart } from "@mantine/charts";
import { DataTable } from "mantine-datatable";
import { IconFilterOff, IconRefresh } from "@tabler/icons-react";
import type { TooltipContentProps } from "recharts";
import { ActiveFilterChips, type FilterChip } from "@/components/ActiveFilterChips";
import { CursorPagination } from "@/components/CursorPagination";
import { useAddressHistory } from "../hooks/useAddressHistory";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";
import { getAddressHistoryColumns } from "./addressHistoryColumns";
import { SOURCE_LABELS } from "../constants";
import { ErrorState } from "@/components/ErrorState";
import { formatChartLabel, presetToMs } from "@/lib/formatChartLabel";
import { useDateFormatter, usePickerValueFormat } from "@/contexts/useDateTimePrefs";
import { useDeviceList } from "@/features/devices/hooks/useDeviceList";
import { useFilterButtonLabels } from "@/hooks/useFilterButtonLabels";
import dayjs from "dayjs";

const PAGE_SIZE = 25;

function GapTooltip({ active, payload, label }: TooltipContentProps<number, string>) {
    if (!active || !payload?.length) return null;
    const item = payload[0];
    if (!item) return null;
    const gapCount = (item.payload as { gap_count?: number }).gap_count ?? 0;
    return (
        <Paper withBorder shadow="sm" p={6} radius="sm">
            <Text size="xs" c="dimmed" mb={2}>{label}</Text>
            <Text size="sm">{item.value} distinct IP{item.value !== 1 ? "s" : ""}</Text>
            {gapCount > 0 && (
                <Text size="xs" c="var(--pw-amber-text)" mt={2}>
                    {gapCount} address{gapCount !== 1 ? "es" : ""} expired this period
                </Text>
            )}
        </Paper>
    );
}

interface AddressHistoryTableProps {
    filters: AddressHistoryFilters;
    refreshInterval: number;
}

export function AddressHistoryTable({ filters, refreshInterval }: AddressHistoryTableProps) {
    const navigate = useNavigate();
    const formatDateTime = useDateFormatter();
    const pickerValueFormat = usePickerValueFormat();

    const [cursor, setCursor] = useState<string | null>(null);
    const [filterKey, setFilterKey] = useState(filters.filterKey);
    if (filterKey !== filters.filterKey) {
        setFilterKey(filters.filterKey);
        setCursor(null);
    }

    const tableRef = useFilterButtonLabels({
        timestamp: "Filter by time",
        device_name: "Filter by device",
        ip: "Filter by IP address",
        is_enabled: "Filter by status",
        source: "Filter by source",
    });

    const { data: ownerGroups } = useDeviceList();

    const { data, isPending, isFetching, error, refetch } = useAddressHistory(
        { ...filters.queryParams, before_id: cursor ? Number(cursor) : undefined, limit: PAGE_SIZE },
        refreshInterval === 0 ? false : refreshInterval,
    );

    const rows = data?.events ?? [];

    const deviceOptions = (ownerGroups ?? []).flatMap((g) => g.devices).map((d) => ({ value: String(d.id), label: d.name }));

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
        onDeviceClick: (deviceId) => {
          const ownerId = (ownerGroups ?? []).find((g) =>
            g.devices.some((d) => d.id === deviceId)
          )?.owner.id;
          if (ownerId !== undefined) navigate(`${buildRoute.userDevices(ownerId)}?device=${deviceId}`);
        },
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

    // Chart data from buckets — use shared formatter
    const timeRangeMs = useMemo(() => {
        if (filters.presetStr) return presetToMs(filters.presetStr);
        if (filters.fromStr && filters.toStr) {
            return dayjs(filters.toStr).diff(dayjs(filters.fromStr));
        }
        return presetToMs("last_24h");
    }, [filters.presetStr, filters.fromStr, filters.toStr]);

    const chartData = useMemo(() => {
        if (!data?.buckets) return [];
        return data.buckets.map((b) => ({
            timestamp: formatChartLabel(b.timestamp, timeRangeMs),
            active_count: b.active_count,
            gap_count: b.gap_count,
        }));
    }, [data, timeRangeMs]);

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
        return <ErrorState error={error} onRetry={() => refetch()} />;
    }

    const total = data?.total_events ?? 0;

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
                        yAxisProps={{ allowDecimals: false }}
                        tooltipProps={{ content: GapTooltip }}
                    />
                ) : (
                    <Text size="sm" c="dimmed" ta="center" py="xl">
                        No activity in this period
                    </Text>
                )}
            </Card>

            <Group justify="flex-end" gap="xs">
                <Tooltip label="Refresh" withArrow>
                    <ActionIcon
                        variant="subtle"
                        color="gray"
                        size="md"
                        onClick={() => refetch()}
                        loading={isFetching}
                        aria-label="Refresh"
                    >
                        <IconRefresh size={16} />
                    </ActionIcon>
                </Tooltip>
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

            <div ref={tableRef} aria-busy={isFetching}>
                <DataTable
                    records={rows}
                    idAccessor="id"
                    highlightOnHover
                    minHeight={150}
                    noRecordsText="No address events found."
                    columns={columns}
                    fetching={isFetching}
                    loaderBackgroundBlur={1}
                    scrollAreaProps={{ type: "auto" }}
                    rowStyle={(r) => (r.is_refresh ? { opacity: 0.55 } : undefined)}
                />
            </div>

            <CursorPagination
                total={total}
                nextCursor={data?.next_cursor != null ? String(data.next_cursor) : null}
                pageSize={PAGE_SIZE}
                onCursorChange={setCursor}
                resetKey={filters.filterKey}
            />
        </Stack>
    );
}
