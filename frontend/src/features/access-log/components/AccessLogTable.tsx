import { useCallback, useEffect, useMemo, useState } from "react";
import { buildRoute } from "@/lib/routes";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ActionIcon, Anchor, Button, Checkbox, Group, Menu, Skeleton, Stack, Text, Tooltip } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { DataTable, type DataTableSortStatus, useDataTableColumns } from "mantine-datatable";
import { IconColumns3, IconDatabaseOff, IconFilterOff, IconRefresh, IconRestore } from "@tabler/icons-react";
import type { AccessLogRow } from "@/lib/api";
import { ActiveFilterChips, type FilterChip } from "@/components/ActiveFilterChips";
import { CursorPagination } from "@/components/CursorPagination";
import { TrafficLineChart } from "@/components/TrafficLineChart";
import { presetToMs } from "@/lib/formatChartLabel";
import { useAccessLog } from "../hooks/useAccessLog";
import { useDashboardTraffic } from "@/features/dashboard/hooks/useDashboardTraffic";
import type { AccessLogFilters } from "../hooks/useAccessLogFilters";
import { AccessLogDetailDrawer } from "./AccessLogDetailDrawer";
import { getAccessLogColumns } from "./accessLogColumns";
import { DENY_REASON_LABELS } from "../constants";
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
import { useDeviceList } from "@/features/devices/hooks/useDeviceList";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useAccessLogDenyReasons } from "../hooks/useAccessLogDenyReasons";
import { useNetworkPolicies } from "@/features/network-policies/hooks/useNetworkPolicies";
import { useFilterButtonLabels } from "@/hooks/useFilterButtonLabels";
import classes from "./AccessLogTable.module.css";

interface AccessLogTableProps {
    filters: AccessLogFilters;
    refreshInterval: number;
}

const PAGE_SIZE = 25;

/** Narrowest a user may resize a column before its header controls get clipped. */
const MIN_COLUMN_WIDTH = 110;

/**
 * Every data column the chooser can show, in display order. Time is mandatory —
 * always shown and not toggleable, since it anchors the pinned first column and
 * the default sort. `defaultVisible` sets the initial state for the rest before
 * the user customises the chooser. The trailing actions column is always
 * rendered and is not listed here.
 */
const COLUMN_META: { accessor: string; label: string; mandatory?: boolean; defaultVisible?: boolean }[] = [
    { accessor: "created_at", label: "Time", mandatory: true },
    { accessor: "client_ip", label: "IP", defaultVisible: true },
    { accessor: "country_code", label: "Country", defaultVisible: true },
    { accessor: "target_host", label: "Host", defaultVisible: true },
    { accessor: "target_uri", label: "URI" },
    { accessor: "http_method", label: "Method" },
    { accessor: "user_id", label: "User", defaultVisible: true },
    { accessor: "authorized_by", label: "Authorized by", defaultVisible: true },
    { accessor: "outcome", label: "Outcome", defaultVisible: true },
    { accessor: "deny_reason", label: "Reason", defaultVisible: true },
    { accessor: "duration_us", label: "Duration", defaultVisible: true },
];

const MANDATORY_COLUMNS = new Set(COLUMN_META.filter((c) => c.mandatory).map((c) => c.accessor));
const DEFAULT_VISIBLE_COLUMNS = COLUMN_META.filter((c) => !c.mandatory && c.defaultVisible).map((c) => c.accessor);
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
const COLUMNS_STORE_KEY = "pulseweaver:access-log:columns:v3";

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
    // `selectedRow` caches the clicked row so its content stays rendered through
    // the close animation after the param is cleared, and is the fallback when a
    // back/forward navigation lands on a request that's no longer on the page.
    const [searchParams, setSearchParams] = useSearchParams();
    const requestParam = searchParams.get("request");
    const [selectedRow, setSelectedRow] = useState<AccessLogRow | null>(null);

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
        outcome: "Filter by outcome",
        deny_reason: "Filter by deny reason",
    });

    const { data: ownerGroups } = useDeviceList();
    const { data: users } = useListUsers();
    const { data: denyReasons } = useAccessLogDenyReasons();
    const { data: networkPolicies } = useNetworkPolicies();

    const { data, isPending, isFetching, error, refetch } = useAccessLog(
        { ...filters.queryParams, cursor: cursor ?? undefined, limit: PAGE_SIZE },
        refreshInterval === 0 ? false : refreshInterval,
    );

    const timeRangeMs = filters.presetStr ? presetToMs(filters.presetStr) : 0;
    const { data: trafficData, isLoading: trafficLoading } = useDashboardTraffic(
        filters.queryParams?.from,
        filters.queryParams?.to,
    );

    const rows = data?.rows ?? [];

    const drawerRow =
        (requestParam != null
            ? rows.find((r) => String(r.id) === requestParam)
            : undefined) ?? selectedRow;

    const openRequest = useCallback(
        (row: AccessLogRow) => {
            setSelectedRow(row);
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

    const deviceOptions = (ownerGroups ?? []).flatMap((g) => g.devices).map((d) => ({ value: String(d.id), label: d.name }));
    const userOptions = (users ?? []).map((u) => ({ value: String(u.id), label: u.display_name || u.username }));
    const denyReasonOptions = (denyReasons ?? []).map((r) => ({
        value: r,
        label: DENY_REASON_LABELS[r] ?? r,
    }));
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

    const defaultVisible = new Set(isCompact ? LEAN_DEFAULT_VISIBLE_COLUMNS : DEFAULT_VISIBLE_COLUMNS);
    const managedColumns = allColumns.map((col) => {
        const accessor = String(col.accessor);
        if (accessor === "actions") return col;
        const mandatory = MANDATORY_COLUMNS.has(accessor);
        return {
            ...col,
            resizable: true,
            // Time is left non-draggable so the statically pinned first column
            // stays locked at index 0. The drag-handle and hide icons are removed
            // (CSS / toggleable) to declutter; reorder still works by dragging the
            // header itself, and visibility is driven from the Columns chooser.
            draggable: accessor !== "created_at",
            toggleable: false,
            defaultToggle: mandatory || defaultVisible.has(accessor),
        };
    });

    const {
        effectiveColumns,
        columnsToggle,
        setColumnsToggle,
        columnsWidth,
        setColumnsWidth,
        resetColumnsOrder,
        resetColumnsToggle,
        resetColumnsWidth,
    } = useDataTableColumns<AccessLogRow>({
        key: COLUMNS_STORE_KEY,
        columns: managedColumns,
        getInitialValueInEffect: false,
    });

    // The resize feature switches the table to `table-layout: fixed` (where CSS
    // min-width is ignored) and its own drag floor of 50px is low enough to clip
    // the sort/filter/hide controls out of a header. The width store is the only
    // input the table honours, so clamp it there. The debounce lets an
    // in-progress drag run unhindered — an undersized column snaps back to the
    // floor on release. The fixed-width actions column is exempt.
    useEffect(() => {
        const timer = setTimeout(() => {
            const updates: { accessor: string; width: string }[] = [];
            for (const entry of columnsWidth) {
                for (const [accessor, width] of Object.entries(entry)) {
                    if (accessor === "actions" || typeof width !== "string") continue;
                    const px = parseInt(width, 10);
                    if (!Number.isNaN(px) && px < MIN_COLUMN_WIDTH) {
                        updates.push({ accessor, width: `${MIN_COLUMN_WIDTH}px` });
                    }
                }
            }
            if (updates.length > 0) setColumnsWidth(updates);
        }, 200);
        return () => clearTimeout(timer);
    }, [columnsWidth, setColumnsWidth]);

    const columnVisible = new Map(columnsToggle.map((c) => [c.accessor, c.toggled]));

    function setColumnVisible(accessor: string, visible: boolean) {
        setColumnsToggle(columnsToggle.map((c) => (c.accessor === accessor ? { ...c, toggled: visible } : c)));
    }

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
            <Stack gap="xs">
                {Array.from({ length: 5 }).map((_, i) => (
                    <Skeleton key={i} height={40} radius="sm" />
                ))}
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
                    data={trafficData?.buckets}
                    isLoading={trafficLoading}
                    timeRangeMs={timeRangeMs || 24 * 60 * 60 * 1000}
                    h={200}
                />

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
                    <Menu shadow="md" closeOnItemClick={false} position="bottom-end">
                        <Menu.Target>
                            <Button
                                variant="subtle"
                                size="compact-xs"
                                leftSection={<IconColumns3 size={14} />}
                            >
                                Columns
                            </Button>
                        </Menu.Target>
                        <Menu.Dropdown>
                            <Menu.Label>Columns</Menu.Label>
                            <Stack gap="xs" px="sm" py={4}>
                                {COLUMN_META.map((c) => (
                                    <Checkbox
                                        key={c.accessor}
                                        size="xs"
                                        label={c.label}
                                        checked={c.mandatory || (columnVisible.get(c.accessor) ?? false)}
                                        disabled={c.mandatory}
                                        onChange={(e) => setColumnVisible(c.accessor, e.currentTarget.checked)}
                                    />
                                ))}
                            </Stack>
                            <Menu.Divider />
                            <Menu.Item
                                leftSection={<IconRestore size={14} />}
                                onClick={() => {
                                    resetColumnsOrder();
                                    resetColumnsToggle();
                                    resetColumnsWidth();
                                }}
                            >
                                Reset columns
                            </Menu.Item>
                            <Menu.Divider />
                            <Text size="xs" c="dimmed" px="sm" py={4}>
                                Tip: drag a column header to reorder.
                            </Text>
                        </Menu.Dropdown>
                    </Menu>
                </Group>

                <ActiveFilterChips chips={filterChips} onClearAll={filters.clearAll} />

                <div ref={tableRef} aria-busy={isFetching} className={classes.table}>
                    <DataTable
                        records={rows}
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
                row={drawerRow}
                opened={requestParam != null}
                onClose={closeRequest}
            />
        </>
    );
}
