import { useState, useMemo } from "react";
import { buildRoute } from "@/lib/routes";
import { useNavigate } from "react-router-dom";
import { ActionIcon, Card, Group, Skeleton, Stack, Text, Tooltip } from "@mantine/core";
import { LineChart } from "@mantine/charts";
import { DataTable } from "mantine-datatable";
import { IconRefresh } from "@tabler/icons-react";
import { ActiveFilterChips, type FilterChip } from "@/components/ActiveFilterChips";
import { CursorPagination } from "@/components/CursorPagination";
import { useAddressHistory } from "../hooks/useAddressHistory";
import { useAddressHistoryHistogram } from "../hooks/useAddressHistoryHistogram";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";
import { getAddressHistoryColumns } from "./addressHistoryColumns";
import {
    COLUMN_CHIP_LABELS,
    FILTER_COLUMN_KEYS,
    type FilterColumnKey,
    describeColumnFilter,
    isFilterActive,
} from "../filterConfig";
import { SOURCE_LABELS, EVENT_KIND_LABELS, TTL_RISK_LABELS, isAddressEventSource, isStateChangesOnly } from "../constants";
import { AddressEventKind, type TtlRisk } from "@/lib/api";
import { ErrorState } from "@/components/ErrorState";
import { formatChartLabel, presetToMs } from "@/lib/formatChartLabel";
import { useDateFormatter, usePickerValueFormat } from "@/contexts/useDateTimePrefs";
import { useDeviceRefs } from "@/features/devices/hooks/useDeviceRefs";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useFilterButtonLabels } from "@/hooks/useFilterButtonLabels";
import dayjs from "dayjs";

const PAGE_SIZE = 25;

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
        device_name: "Filter by device or owning user",
        ip: "Filter by IP address",
        source: "Filter by source",
        event_kind: "Filter by event kind",
        ttl_risk: "Filter by TTL risk",
    });

    const { data: deviceRefs } = useDeviceRefs();
    const { data: users } = useListUsers();

    const refetchIntervalOrFalse = refreshInterval === 0 ? false : refreshInterval;

    // Distinct query keys by construction: the events key includes the cursor
    // (before_id/limit), the histogram key never does — so a page turn only
    // ever invalidates the events query.
    const { data, isPending, isFetching, error, refetch } = useAddressHistory(
        { ...filters.queryParams, before_id: cursor ? Number(cursor) : undefined, limit: PAGE_SIZE },
        refetchIntervalOrFalse,
    );

    const {
        data: histogramData,
        isFetching: isHistogramFetching,
        refetch: refetchHistogram,
    } = useAddressHistoryHistogram(filters.queryParams, refetchIntervalOrFalse);

    const rows = data?.events ?? [];

    const deviceOptions = (deviceRefs ?? []).map((d) => ({ value: String(d.id), label: d.name }));
    const userOptions = (users ?? []).map((u) => ({ value: String(u.id), label: u.display_name || u.username }));

    const columns = getAddressHistoryColumns({
        formatDateTime,
        pickerValueFormat,
        fromStr: filters.fromStr,
        toStr: filters.toStr,
        setSearchParams: filters.setSearchParams,
        getColumnFilter: filters.getColumnFilter,
        setColumnFilter: filters.setColumnFilter,
        lockedFilterKey: filters.lockedFilter?.key,
        deviceOptions,
        userOptions,
        onDeviceClick: (deviceId) => {
            const ownerId = (deviceRefs ?? []).find((d) => d.id === deviceId)?.owner_id;
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

        const resolvers: Partial<Record<FilterColumnKey, (v: string) => string>> = {
            device_id: (v) => deviceOptions.find((o) => o.value === v)?.label ?? v,
            user_id: (v) => userOptions.find((o) => o.value === v)?.label ?? v,
            source: (v) => (isAddressEventSource(v) ? SOURCE_LABELS[v] : v),
            event_kind: (v) => EVENT_KIND_LABELS[v as AddressEventKind] ?? v,
            ttl_risk: (v) => TTL_RISK_LABELS[v as TtlRisk] ?? v,
        };

        for (const key of FILTER_COLUMN_KEYS) {
            if (filters.lockedFilter?.key === key) continue;
            const state = filters.getColumnFilter(key);
            if (!isFilterActive(state)) continue;
            if (key === "event_kind" && isStateChangesOnly(state)) continue;
            chips.push({
                label: COLUMN_CHIP_LABELS[key],
                value: describeColumnFilter(key, state, resolvers[key]),
                onRemove: () => filters.setColumnFilter(key, null),
            });
        }

        return chips;
    }, [filters, formatDateTime, deviceOptions, userOptions]);

    // Chart data from buckets — use shared formatter
    const timeRangeMs = useMemo(() => {
        if (filters.presetStr) return presetToMs(filters.presetStr);
        if (filters.fromStr && filters.toStr) {
            return dayjs(filters.toStr).diff(dayjs(filters.fromStr));
        }
        return presetToMs("last_24h");
    }, [filters.presetStr, filters.fromStr, filters.toStr]);

    const chartData = useMemo(() => {
        if (!histogramData?.buckets) return [];
        return histogramData.buckets.map((b) => ({
            timestamp: formatChartLabel(b.timestamp, timeRangeMs),
            event_count: b.event_count,
        }));
    }, [histogramData, timeRangeMs]);

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

    const total = data?.total ?? 0;

    return (
        <Stack gap="sm">
            <Card withBorder padding="md">
                <Text fw={500} mb="sm">Events over time</Text>
                {chartData.length > 0 ? (
                    <LineChart
                        h={200}
                        data={chartData}
                        dataKey="timestamp"
                        series={[{ name: "event_count", color: "orange.4", label: "Events" }]}
                        yAxisLabel="Events"
                        curveType="monotone"
                        tooltipAnimationDuration={150}
                        yAxisProps={{ allowDecimals: false }}
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
                        onClick={() => {
                            refetch();
                            refetchHistogram();
                        }}
                        loading={isFetching || isHistogramFetching}
                        aria-label="Refresh"
                    >
                        <IconRefresh size={16} />
                    </ActionIcon>
                </Tooltip>
            </Group>

            <ActiveFilterChips chips={filterChips} onClearAll={filters.clearAll} />

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
                    pinFirstColumn
                    rowStyle={(r) => (r.event_kind === AddressEventKind.REFRESH ? { opacity: 0.55 } : undefined)}
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
