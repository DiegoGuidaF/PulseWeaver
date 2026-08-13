import { useCallback, useMemo, useState } from "react";
import dayjs from "dayjs";
import { buildRoute } from "@/lib/routes";
import { useNavigate, useSearchParams } from "react-router";
import { ActionIcon, Anchor, Button, Group, Skeleton, Stack, Text, Tooltip } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { DataTable, type DataTableSortStatus } from "mantine-datatable";
import { IconDatabaseOff, IconFilterOff, IconRefresh } from "@tabler/icons-react";
import type { AccessLogRow } from "@/lib/api";
import { ActiveFilterChips, type FilterChip } from "@/components/ActiveFilterChips";
import { ColumnsMenu } from "@/components/ColumnsMenu";
import { useManagedColumns, type ManagedColumnMeta } from "@/hooks/useManagedColumns";
import { CursorPagination } from "@/components/CursorPagination";
import { TrafficLineChart } from "@/components/TrafficLineChart";
import { presetToMs } from "@/lib/formatChartLabel";
import { useAccessLog } from "../hooks/useAccessLog";
import { useAccessLogHistogram } from "../hooks/useAccessLogHistogram";
import type { AccessLogFilters } from "../hooks/useAccessLogFilters";
import { AccessLogDetailDrawer } from "./AccessLogDetailDrawer";
import { getAccessLogColumns } from "./accessLogColumns";
import { POLICY_DENY_REASON_OPTIONS } from "@/lib/policyDenyReasons";
import {
    type FilterColumnKey,
    type SortColumn,
    COLUMN_CHIP_LABELS,
    FILTER_COLUMN_KEYS,
    describeColumnFilter,
    isFilterActive,
    nextSortState,
} from "../filterConfig";
import { ErrorState } from "@/components/ErrorState";
import { useDateFormatter, usePickerValueFormat } from "@/contexts/useDateTimePrefs";
import { useDeviceRefs } from "@/features/devices/hooks/useDeviceRefs";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useNetworkPolicies } from "@/features/network-policies/hooks/useNetworkPolicies";
import { useFilterButtonLabels } from "@/hooks/useFilterButtonLabels";
import classes from "@/components/managedColumns.module.css";

interface AccessLogTableProps {
    filters: AccessLogFilters;
    refreshInterval: number;
}

const PAGE_SIZE = 25;

/**
 * Fixed table viewport so the body scrolls internally with a sticky header. This
 * keeps the horizontal scrollbar pinned to the bottom of the visible table
 * instead of stranding it below a full page of rows.
 */
const TABLE_HEIGHT = 560;

/** Narrowest a user may resize a column before its header controls get clipped. */
const MIN_COLUMN_WIDTH = 110;

/**
 * Every data column the chooser can show, in display order. Time is mandatory —
 * always shown and not toggleable, since it anchors the pinned first column and
 * the default sort. `defaultVisible` sets the initial state for the rest before
 * the user customises the chooser. The trailing actions column is always
 * rendered and is not listed here.
 */
const COLUMN_META: ManagedColumnMeta[] = [
    { accessor: "created_at", label: "Time", mandatory: true },
    { accessor: "client_ip", label: "IP", defaultVisible: true },
    { accessor: "country_code", label: "Country", defaultVisible: true },
    { accessor: "target_host", label: "Host", defaultVisible: true },
    { accessor: "target_uri", label: "URI" },
    { accessor: "http_method", label: "Method" },
    { accessor: "user_id", label: "User", defaultVisible: true },
    { accessor: "authorized_by", label: "Authorized by" },
    { accessor: "outcome", label: "Decision", defaultVisible: true },
    { accessor: "duration_us", label: "Duration" },
];

/**
 * Compact default for screens below `md`: the identifying IP/Host and the
 * headline Outcome alongside the mandatory Time, so the table fits without
 * horizontal scrolling. Seeds `defaultToggle` on first visit only — a stored
 * column choice wins at any width.
 */
const LEAN_DEFAULT_VISIBLE_COLUMNS = ["client_ip", "target_host", "outcome"];

/**
 * Key for the mantine-datatable column store. The library persists column order,
 * visibility and width under `${key}-columns-*`, keeping the chooser, drag-to-reorder
 * and resize handles in sync through one store.
 */
const COLUMNS_STORE_KEY = "pulseweaver:access-log:columns:v4";

export function AccessLogTable({ filters, refreshInterval }: AccessLogTableProps) {
    const navigate = useNavigate();
    const formatDateTime = useDateFormatter();
    const pickerValueFormat = usePickerValueFormat();

    const [cursor, setCursor] = useState<string | null>(null);

    // Reset cursor when filters or sort change (the cursor encodes the sort).
    const [filterKey, setFilterKey] = useState(filters.filterKey);
    if (filterKey !== filters.filterKey) {
        setFilterKey(filters.filterKey);
        setCursor(null);
    }

    // The detail drawer's open state lives in the URL (`?request=<id>`) so the
    // browser Back button closes it — the expected gesture, especially on mobile.
    // The drawer fetches the request by id, so a deep link resolves even when the
    // row sits on a page the table has not loaded.
    const [searchParams, setSearchParams] = useSearchParams();
    const requestParam = searchParams.get("request");
    const requestId = requestParam != null ? Number(requestParam) : null;

    // Below the nav-collapse breakpoint, start from a lean column set to avoid
    // horizontal scrolling. Matches the AppShell's `md` threshold.
    const isCompact = !useMediaQuery("(min-width: 62em)", true, { getInitialValueInEffect: false });

    const tableRef = useFilterButtonLabels({
        created_at: "Filter by time",
        client_ip: "Filter by IP address",
        country_code: "Filter by country",
        target_host: "Filter by host",
        target_uri: "Filter by URI",
        http_method: "Filter by HTTP method",
        user_id: "Filter by user",
        authorized_by: "Filter by authorized device or policy",
        outcome: "Filter by decision",
    });

    const { data: deviceRefs } = useDeviceRefs();
    const { data: users } = useListUsers();
    const { data: networkPolicies } = useNetworkPolicies();

    const refetchIntervalOrFalse = refreshInterval === 0 ? false : refreshInterval;

    // Distinct query keys by construction: the list key carries sort and cursor,
    // the histogram key carries neither — so a page turn or a sort change never
    // re-scans the chart, and both read the same filters over the same window.
    const { data, isPending, isFetching, error, refetch } = useAccessLog(
        {
            ...filters.queryParams,
            sort: filters.sort,
            order: filters.order,
            cursor: cursor ?? undefined,
            limit: PAGE_SIZE,
        },
        refetchIntervalOrFalse,
    );

    const {
        data: histogramData,
        isPending: isHistogramPending,
        isFetching: isHistogramFetching,
        error: histogramError,
        refetch: refetchHistogram,
    } = useAccessLogHistogram(filters.queryParams, refetchIntervalOrFalse);

    const timeRangeMs = useMemo(() => {
        if (filters.presetStr) return presetToMs(filters.presetStr);
        if (filters.fromStr && filters.toStr) {
            return dayjs(filters.toStr).diff(dayjs(filters.fromStr));
        }
        return presetToMs("last_24h");
    }, [filters.presetStr, filters.fromStr, filters.toStr]);

    const rows = data?.rows ?? [];

    const openRequest = useCallback(
        (row: AccessLogRow) => {
            setSearchParams((prev) => {
                const next = new URLSearchParams(prev);
                next.set("request", String(row.id));
                return next;
            });
        },
        [setSearchParams],
    );

    const closeRequest = useCallback(() => {
        // Replace rather than push so the cleared state doesn't add a history
        // entry that Back would step into and re-open the drawer.
        setSearchParams(
            (prev) => {
                const next = new URLSearchParams(prev);
                next.delete("request");
                return next;
            },
            { replace: true },
        );
    }, [setSearchParams]);

    const deviceOptions = (deviceRefs ?? []).map((d) => ({ value: String(d.id), label: d.name }));
    const userOptions = (users ?? []).map((u) => ({ value: String(u.id), label: u.display_name || u.username }));
    const denyReasonOptions = POLICY_DENY_REASON_OPTIONS;
    const networkPolicyOptions = (networkPolicies ?? []).map((p) => ({
        value: String(p.id),
        label: `${p.name} (${p.cidr})`,
    }));

    const allColumns = getAccessLogColumns({
        formatDateTime,
        pickerValueFormat,
        fromStr: filters.fromStr,
        toStr: filters.toStr,
        outcomeStr: filters.outcomeStr,
        setOutcome: filters.setOutcome,
        getColumnFilter: filters.getColumnFilter,
        setColumnFilter: filters.setColumnFilter,
        setSearchParams: filters.setSearchParams,
        deviceOptions,
        denyReasonOptions,
        networkPolicyOptions,
        userOptions,
        onRowClick: openRequest,
        onUserClick: (userId) => navigate(buildRoute.userDevices(userId)),
        onDeviceClick: (deviceId, ownerUserId) => {
            if (ownerUserId !== undefined) navigate(`${buildRoute.userDevices(ownerUserId)}?device=${deviceId}`);
        },
        onNetworkPolicyClick: (id) => navigate(buildRoute.accessNetworkPolicyDetail(id)),
    });

    const { effectiveColumns, columnVisible, setColumnVisible, resetColumns } =
        useManagedColumns<AccessLogRow>({
            storeKey: COLUMNS_STORE_KEY,
            columns: allColumns,
            meta: COLUMN_META,
            compactVisible: LEAN_DEFAULT_VISIBLE_COLUMNS,
            compact: isCompact,
            minColumnWidth: MIN_COLUMN_WIDTH,
        });

    const sortStatus: DataTableSortStatus<AccessLogRow> = {
        columnAccessor: filters.sort,
        direction: filters.order,
    };

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

        if (filters.outcomeStr) {
            chips.push({
                label: "Outcome",
                value: filters.outcomeStr === "allow" ? "Allow" : "Deny",
                onRemove: () => filters.setOutcome(null),
            });
        }

        const resolvers: Partial<Record<FilterColumnKey, (v: string) => string>> = {
            device_id: (v) => deviceOptions.find((o) => o.value === v)?.label ?? v,
            user_id: (v) => userOptions.find((o) => o.value === v)?.label ?? v,
            network_policy_id: (v) => networkPolicyOptions.find((o) => o.value === v)?.label ?? v,
            deny_reason: (v) => denyReasonOptions.find((o) => o.value === v)?.label ?? v,
        };

        for (const key of FILTER_COLUMN_KEYS) {
            const state = filters.getColumnFilter(key);
            if (!isFilterActive(state)) continue;
            chips.push({
                label: COLUMN_CHIP_LABELS[key],
                value: describeColumnFilter(key, state, resolvers[key]),
                onRemove: () => filters.setColumnFilter(key, null),
            });
        }

        return chips;
    }, [filters, formatDateTime, deviceOptions, userOptions, networkPolicyOptions, denyReasonOptions]);

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
        <>
            <Stack gap="sm">
                <TrafficLineChart
                    data={histogramData?.buckets}
                    isLoading={isHistogramPending}
                    timeRangeMs={timeRangeMs}
                    h={200}
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
                        meta={COLUMN_META}
                        columnVisible={columnVisible}
                        setColumnVisible={setColumnVisible}
                        onReset={resetColumns}
                    />
                </Group>

                <ActiveFilterChips chips={filterChips} onClearAll={filters.clearAll} />

                <div ref={tableRef} aria-busy={isFetching} className={classes.table}>
                    <DataTable
                        records={rows}
                        highlightOnHover
                        height={TABLE_HEIGHT}
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
                                        No matching log entries.
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
                        sortStatus={sortStatus}
                        onSortStatusChange={(status) => {
                            const next = nextSortState(
                                { sort: filters.sort, order: filters.order },
                                status.columnAccessor as SortColumn,
                            );
                            filters.setSort(next.sort, next.order);
                        }}
                    />
                </div>

                <CursorPagination
                    total={total}
                    nextCursor={data?.next_cursor ?? null}
                    pageSize={PAGE_SIZE}
                    onCursorChange={setCursor}
                    resetKey={filters.filterKey}
                />

                {rows.some((r) => r.country_code) && (
                    <Text size="xs" c="dimmed" ta="right">
                        <Anchor href="https://db-ip.com" target="_blank" rel="noopener noreferrer" size="xs" c="dimmed">
                            IP Geolocation by DB-IP
                        </Anchor>
                    </Text>
                )}
            </Stack>

            <AccessLogDetailDrawer
                requestId={requestId}
                opened={requestParam != null}
                onClose={closeRequest}
            />
        </>
    );
}
