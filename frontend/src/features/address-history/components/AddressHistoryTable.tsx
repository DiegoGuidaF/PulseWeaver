import { useState, useMemo } from "react";
import { buildRoute } from "@/lib/routes";
import { useNavigate } from "react-router";
import { ActionIcon, Button, Group, Skeleton, Stack, Text, Tooltip } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { DataTable } from "mantine-datatable";
import { IconDatabaseOff, IconFilterOff, IconRefresh } from "@tabler/icons-react";
import { ActiveFilterChips, type FilterChip } from "@/components/ActiveFilterChips";
import { ColumnsMenu } from "@/components/ColumnsMenu";
import { useManagedColumns, type ManagedColumnMeta } from "@/hooks/useManagedColumns";
import { CursorPagination } from "@/components/CursorPagination";
import { useAddressHistory } from "../hooks/useAddressHistory";
import { useAddressHistoryHistogram } from "../hooks/useAddressHistoryHistogram";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";
import { getAddressHistoryColumns } from "./addressHistoryColumns";
import { AddressHistoryRiskChart } from "./AddressHistoryRiskChart";
import {
    COLUMN_CHIP_LABELS,
    FILTER_COLUMN_KEYS,
    type FilterColumnKey,
    describeColumnFilter,
    isFilterActive,
} from "../filterConfig";
import {
    EVENT_KIND_LABELS,
    SOURCE_LABELS,
    TTL_HEADROOM_COLUMN_LABEL,
    TTL_RISK_LABELS,
    isAddressEventSource,
} from "../constants";
import { AddressEventKind, type AddressHistoryEvent, type TtlRisk } from "@/lib/api";
import { ErrorState } from "@/components/ErrorState";
import { presetToMs } from "@/lib/formatChartLabel";
import { useDateFormatter, usePickerValueFormat } from "@/contexts/useDateTimePrefs";
import { useDeviceRefs } from "@/features/devices/hooks/useDeviceRefs";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useFilterButtonLabels } from "@/hooks/useFilterButtonLabels";
import classes from "@/components/managedColumns.module.css";
import dayjs from "dayjs";

const PAGE_SIZE = 25;

/**
 * Every data column the chooser can show, in display order. Time is mandatory —
 * always shown and not toggleable, since it anchors the pinned first column and
 * the fixed `id DESC` order. `defaultVisible` seeds the rest on first visit.
 */
const COLUMN_META: ManagedColumnMeta[] = [
    { accessor: "timestamp", label: "Time", mandatory: true },
    { accessor: "device_name", label: "Device", defaultVisible: true },
    { accessor: "ip", label: "IP", defaultVisible: true },
    { accessor: "geo", label: "Location", defaultVisible: true },
    { accessor: "is_enabled", label: "Status", defaultVisible: true },
    { accessor: "source", label: "Source", defaultVisible: true },
    { accessor: "event_kind", label: "Event", defaultVisible: true },
    { accessor: "renewal_gap_seconds", label: "Renewal gap", defaultVisible: true },
    { accessor: "ttl_risk", label: TTL_HEADROOM_COLUMN_LABEL, defaultVisible: true },
];

/**
 * Compact default below `md`: what identifies the event (IP, Event) plus the
 * headline health reading, so the table fits without horizontal scrolling.
 */
const LEAN_DEFAULT_VISIBLE_COLUMNS = ["ip", "event_kind", "ttl_risk"];

/** Column store key; bump the suffix when the persisted column shape changes. */
const COLUMNS_STORE_KEY = "pulseweaver:address-history:columns:v1";

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
        ttl_risk: "Filter by TTL headroom",
    });

    // Below the nav-collapse breakpoint, start from a lean column set to avoid
    // horizontal scrolling. Matches the AppShell's `md` threshold.
    const isCompact = !useMediaQuery("(min-width: 62em)", true, { getInitialValueInEffect: false });

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
        isPending: isHistogramPending,
        isFetching: isHistogramFetching,
        error: histogramError,
        refetch: refetchHistogram,
    } = useAddressHistoryHistogram(filters.queryParams, refetchIntervalOrFalse);

    const rows = data?.events ?? [];

    const deviceOptions = (deviceRefs ?? []).map((d) => ({ value: String(d.id), label: d.name }));
    const userOptions = (users ?? []).map((u) => ({ value: String(u.id), label: u.display_name || u.username }));

    const allColumns = getAddressHistoryColumns({
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
        onDeviceClick: goToDevice,
    });

    /** Opens the device workspace on its default tab. */
    function goToDevice(deviceId: number) {
        const ownerId = (deviceRefs ?? []).find((d) => d.id === deviceId)?.owner_id;
        if (ownerId !== undefined) navigate(`${buildRoute.userDevices(ownerId)}?device=${deviceId}`);
    }

    // A locked filter drops the column that displays it, so the chooser must not
    // offer it either — otherwise it lists a column that can never appear.
    const columnMeta = useMemo(() => {
        const rendered = new Set(allColumns.map((c) => String(c.accessor)));
        return COLUMN_META.filter((m) => rendered.has(m.accessor));
    }, [allColumns]);

    const { effectiveColumns, columnVisible, setColumnVisible, resetColumns } =
        useManagedColumns<AddressHistoryEvent>({
            storeKey: COLUMNS_STORE_KEY,
            columns: allColumns,
            meta: columnMeta,
            compactVisible: LEAN_DEFAULT_VISIBLE_COLUMNS,
            compact: isCompact,
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
            <AddressHistoryRiskChart
                buckets={histogramData?.buckets}
                timeRangeMs={timeRangeMs}
                isPending={isHistogramPending}
                error={histogramError}
                onRetry={() => refetchHistogram()}
            />

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
                <ColumnsMenu
                    meta={columnMeta}
                    columnVisible={columnVisible}
                    setColumnVisible={setColumnVisible}
                    onReset={resetColumns}
                />
            </Group>

            <ActiveFilterChips chips={filterChips} onClearAll={filters.clearAll} />

            <div ref={tableRef} aria-busy={isFetching} className={classes.table}>
                <DataTable
                    records={rows}
                    idAccessor="id"
                    highlightOnHover
                    minHeight={150}
                    // The empty-state container disables pointer events, so the
                    // clear-filters CTA re-enables them; rendering it only when the
                    // table is actually empty keeps the invisible overlay inert.
                    emptyState={
                        rows.length === 0 ? (
                            <Stack align="center" gap={4} style={{ pointerEvents: "all" }}>
                                <div className="mantine-datatable-empty-state-icon">
                                    <IconDatabaseOff />
                                </div>
                                <Text size="sm" c="dimmed">
                                    No address events found.
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
                            </Stack>
                        ) : undefined
                    }
                    columns={effectiveColumns}
                    storeColumnsKey={COLUMNS_STORE_KEY}
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
