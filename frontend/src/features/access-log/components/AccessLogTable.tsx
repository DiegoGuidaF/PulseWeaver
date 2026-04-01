import { useMemo, useState } from "react";
import { Alert, Anchor, Button, Group, Skeleton, Stack, Text } from "@mantine/core";
import { DataTable } from "mantine-datatable";
import { IconAlertCircle, IconFilterOff } from "@tabler/icons-react";
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
import { toErrorMessage } from "@/lib/api-client";
import { useDateFormatter, usePickerValueFormat } from "@/contexts/useDateTimePrefs";
import { useDevices } from "@/features/devices/hooks/useDevices";
import { useAccessLogDenyReasons } from "../hooks/useAccessLogDenyReasons";

interface AccessLogTableProps {
    filters: AccessLogFilters;
    refreshInterval: number;
}

const PAGE_SIZE = 25;

export function AccessLogTable({ filters, refreshInterval }: AccessLogTableProps) {
    const formatDateTime = useDateFormatter();
    const pickerValueFormat = usePickerValueFormat();

    const [cursor, setCursor] = useState<string | null>(null);

    // Reset cursor when filters change
    const [filterKey, setFilterKey] = useState(filters.filterKey);
    if (filterKey !== filters.filterKey) {
        setFilterKey(filters.filterKey);
        setCursor(null);
    }

    const [selectedRow, setSelectedRow] = useState<AccessLogRow | null>(null);
    const [drawerOpened, setDrawerOpened] = useState(false);

    const { data: devices } = useDevices();
    const { data: denyReasons } = useAccessLogDenyReasons();

    const { data, isPending, error } = useAccessLog(
        { ...filters.queryParams, before_id: cursor ? Number(cursor) : undefined, limit: PAGE_SIZE },
        refreshInterval === 0 ? false : refreshInterval,
    );

    // Chart data — uses the dashboard traffic endpoint with the same time range
    const timeRangeMs = filters.presetStr ? presetToMs(filters.presetStr) : 0;
    const { data: trafficData, isLoading: trafficLoading } = useDashboardTraffic(
        filters.queryParams?.from,
        filters.queryParams?.to,
    );

    const rows = data?.rows ?? [];

    const deviceOptions = (devices ?? []).map((d) => ({ value: String(d.id), label: d.name }));
    const denyReasonOptions = (denyReasons ?? []).map((r) => ({
        value: r,
        label: DENY_REASON_LABELS[r] ?? r,
    }));

    const columns = getAccessLogColumns({
        formatDateTime,
        pickerValueFormat,
        presetStr: filters.presetStr,
        fromStr: filters.fromStr,
        toStr: filters.toStr,
        ipLocal: filters.ipLocal,
        ipDebounced: filters.ipDebounced,
        deviceIdStr: filters.deviceIdStr,
        outcomeStr: filters.outcomeStr,
        denyReason: filters.denyReason,
        countryCodeLocal: filters.countryCodeLocal,
        countryCodeDebounced: filters.countryCodeDebounced,
        deviceOptions,
        denyReasonOptions,
        setParam: filters.setParam,
        setIpLocal: filters.setIpLocal,
        setCountryCodeLocal: filters.setCountryCodeLocal,
        setSearchParams: filters.setSearchParams,
        onRowClick: (row) => {
            setSelectedRow(row);
            setDrawerOpened(true);
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

        if (filters.ipDebounced) {
            chips.push({
                label: "IP",
                value: filters.ipDebounced,
                onRemove: () => filters.setIpLocal(""),
            });
        }

        if (filters.deviceIdStr) {
            const device = deviceOptions.find((d) => d.value === filters.deviceIdStr);
            chips.push({
                label: "Device",
                value: device?.label ?? filters.deviceIdStr,
                onRemove: () => filters.setParam("device_id", null),
            });
        }

        if (filters.outcomeStr) {
            chips.push({
                label: "Outcome",
                value: filters.outcomeStr === "allow" ? "Allow" : "Deny",
                onRemove: () => {
                    filters.setSearchParams((prev) => {
                        prev.delete("outcome");
                        prev.delete("deny_reason");
                        return prev;
                    });
                },
            });
        }

        if (filters.denyReason) {
            chips.push({
                label: "Reason",
                value: DENY_REASON_LABELS[filters.denyReason] ?? filters.denyReason,
                onRemove: () => filters.setParam("deny_reason", null),
            });
        }

        if (filters.countryCodeDebounced) {
            chips.push({
                label: "Country",
                value: filters.countryCodeDebounced,
                onRemove: () => filters.setCountryCodeLocal(""),
            });
        }

        return chips;
    }, [filters, formatDateTime, deviceOptions]);

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

    const total = data?.total ?? 0;

    return (
        <>
            <Stack gap="sm">
                {/* Traffic chart */}
                <TrafficLineChart
                    data={trafficData?.buckets}
                    isLoading={trafficLoading}
                    timeRangeMs={timeRangeMs || 24 * 60 * 60 * 1000}
                    h={200}
                />

                <Group justify="flex-end">
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
                    highlightOnHover
                    minHeight={150}
                    noRecordsText="No matching log entries."
                    columns={columns}
                />

                <CursorPagination
                    total={total}
                    nextCursor={data?.next_cursor != null ? String(data.next_cursor) : null}
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
                row={selectedRow}
                opened={drawerOpened}
                onClose={() => setDrawerOpened(false)}
            />
        </>
    );
}
