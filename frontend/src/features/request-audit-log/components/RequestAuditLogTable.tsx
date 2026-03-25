import { useMemo, useState } from "react";
import { Alert, Button, Group, Skeleton, Stack, Text } from "@mantine/core";
import { DataTable } from "mantine-datatable";
import { IconAlertCircle, IconFilterOff } from "@tabler/icons-react";
import type { RequestAuditLogRow } from "@/lib/api";
import { ActiveFilterChips, type FilterChip } from "@/components/ActiveFilterChips";
import { useRequestAuditLog } from "../hooks/useRequestAuditLog";
import type { AuditLogFilters } from "../hooks/useAuditLogFilters";
import { RequestAuditLogDetailDrawer } from "./RequestAuditLogDetailDrawer";
import { getAuditLogColumns } from "./auditLogColumns";
import { DENY_REASON_LABELS } from "../constants";
import { toErrorMessage } from "@/lib/api-client";
import { useDateFormatter, usePickerValueFormat } from "@/contexts/useDateTimePrefs";
import { useDevices } from "@/features/devices/hooks/useDevices";
import { useRequestAuditLogDenyReasons } from "../hooks/useRequestAuditLogDenyReasons";

interface RequestAuditLogTableProps {
    filters: AuditLogFilters;
    refreshInterval: number;
}

export function RequestAuditLogTable({ filters, refreshInterval }: RequestAuditLogTableProps) {
    const formatDateTime = useDateFormatter();
    const pickerValueFormat = usePickerValueFormat();

    const [pagination, setPagination] = useState({
        filterKey: filters.filterKey,
        beforeId: undefined as number | undefined,
        allRows: [] as RequestAuditLogRow[],
    });
    // Reset pagination when filters change — React's "adjusting state when a
    // prop changes" pattern: setState during render triggers a synchronous
    // re-render before committing, no effect or ref needed.
    if (pagination.filterKey !== filters.filterKey) {
        setPagination({ filterKey: filters.filterKey, beforeId: undefined, allRows: [] });
    }
    const { beforeId, allRows } = pagination;

    const [selectedRow, setSelectedRow] = useState<RequestAuditLogRow | null>(null);
    const [drawerOpened, setDrawerOpened] = useState(false);

    const { data: devices } = useDevices();
    const { data: denyReasons } = useRequestAuditLogDenyReasons();

    const { data, isPending, error } = useRequestAuditLog(
        { ...filters.queryParams, before_id: beforeId },
        refreshInterval === 0 ? false : refreshInterval,
    );

    const rows = beforeId !== undefined
        ? allRows.concat(data?.rows ?? [])
        : (data?.rows ?? []);

    function handleLoadMore() {
        if (!data?.next_cursor) return;
        setPagination((prev) => ({
            ...prev,
            allRows: rows,
            beforeId: data.next_cursor ?? undefined,
        }));
    }

    const deviceOptions = (devices ?? []).map((d) => ({ value: String(d.id), label: d.name }));
    const denyReasonOptions = (denyReasons ?? []).map((r) => ({
        value: r,
        label: DENY_REASON_LABELS[r] ?? r,
    }));

    const columns = getAuditLogColumns({
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
        deviceOptions,
        denyReasonOptions,
        setParam: filters.setParam,
        setIpLocal: filters.setIpLocal,
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
    const hasMore = Boolean(data?.next_cursor);

    return (
        <>
            <Stack gap="sm">
                <Group justify="space-between">
                    <Text size="sm" c="dimmed">
                        {total} result{total !== 1 ? "s" : ""}
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
                    highlightOnHover
                    minHeight={150}
                    noRecordsText="No matching log entries."
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

            <RequestAuditLogDetailDrawer
                row={selectedRow}
                opened={drawerOpened}
                onClose={() => setDrawerOpened(false)}
            />
        </>
    );
}
