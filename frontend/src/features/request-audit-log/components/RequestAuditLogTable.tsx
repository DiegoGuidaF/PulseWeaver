import { useState } from "react";
import { Stack, Text, Badge, Button, Alert, Skeleton, Group, ActionIcon } from "@mantine/core";
import { DataTable } from "mantine-datatable";
import { IconAlertCircle, IconChevronRight } from "@tabler/icons-react";
import type { RequestAuditLogRow } from "@/lib/api";
import type { GetRequestAuditLogData } from "@/lib/api";
import { useRequestAuditLog } from "../hooks/useRequestAuditLog";
import { RequestAuditLogDetailDrawer } from "./RequestAuditLogDetailDrawer";
import { DENY_REASON_LABELS } from "../constants";
import { formatDateTime } from "@/lib/dates";
import { toErrorMessage } from "@/lib/api-client";

interface RequestAuditLogTableProps {
    params: GetRequestAuditLogData["query"];
    refreshInterval: number;
}

export function RequestAuditLogTable({ params, refreshInterval }: RequestAuditLogTableProps) {
    const [beforeId, setBeforeId] = useState<number | undefined>(undefined);
    const [allRows, setAllRows] = useState<RequestAuditLogRow[]>([]);
    const [selectedRow, setSelectedRow] = useState<RequestAuditLogRow | null>(null);
    const [drawerOpened, setDrawerOpened] = useState(false);

    const { data, isPending, error } = useRequestAuditLog(
        { ...params, before_id: beforeId },
        refreshInterval === 0 ? false : refreshInterval,
    );

    // Reset accumulated rows when filters change (beforeId is undefined and data changes)
    const currentRows: RequestAuditLogRow[] = (() => {
        if (!data) return allRows;
        if (beforeId === undefined) {
            // Fresh query — use data directly
            return data.rows;
        }
        return allRows;
    })();

    function handleLoadMore() {
        if (!data?.next_cursor) return;
        setAllRows(displayRows);
        setBeforeId(data.next_cursor);
    }

    function handleRowClick(row: RequestAuditLogRow) {
        setSelectedRow(row);
        setDrawerOpened(true);
    }

    if ((isPending || !data) && currentRows.length === 0) {
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

    const displayRows = beforeId !== undefined ? allRows.concat(data?.rows ?? []) : (data?.rows ?? []);
    const total = data?.total ?? 0;
    const hasMore = Boolean(data?.next_cursor);

    return (
        <>
            <Stack gap="sm">
                <Text size="sm" c="dimmed">
                    {total} result{total !== 1 ? "s" : ""}
                </Text>

                {displayRows.length === 0 ? (
                    <Text c="dimmed" ta="center" py="xl">
                        No matching log entries.
                    </Text>
                ) : (
                    <DataTable
                        records={displayRows}
                        highlightOnHover
                        columns={[
                            {
                                accessor: "created_at",
                                title: "Time",
                                render: (row) => (
                                    <Text size="sm" ff="monospace">
                                        {formatDateTime(row.created_at)}
                                    </Text>
                                ),
                            },
                            {
                                accessor: "client_ip",
                                title: "IP",
                                render: (row) => (
                                    <Text size="sm" ff="monospace">
                                        {row.client_ip}
                                    </Text>
                                ),
                            },
                            {
                                accessor: "target_host",
                                title: "Host",
                                render: (row) => (
                                    <Text size="sm">{row.target_host ?? "—"}</Text>
                                ),
                            },
                            {
                                accessor: "device_name",
                                title: "Device",
                                render: (row) => (
                                    <Text size="sm">{row.device_name ?? "—"}</Text>
                                ),
                            },
                            {
                                accessor: "outcome",
                                title: "Outcome",
                                render: (row) => (
                                    <Badge color={row.outcome ? "green" : "red"} size="sm">
                                        {row.outcome ? "Allow" : "Deny"}
                                    </Badge>
                                ),
                            },
                            {
                                accessor: "deny_reason",
                                title: "Reason",
                                render: (row) =>
                                    row.deny_reason ? (
                                        <Text size="sm">
                                            {DENY_REASON_LABELS[row.deny_reason] ?? row.deny_reason}
                                        </Text>
                                    ) : (
                                        <Text size="sm" c="dimmed">
                                            —
                                        </Text>
                                    ),
                            },
                            {
                                accessor: "actions",
                                title: "",
                                width: 40,
                                render: (row) => (
                                    <ActionIcon
                                        variant="subtle"
                                        color="gray"
                                        size="sm"
                                        onClick={() => handleRowClick(row)}
                                        aria-label="View details"
                                    >
                                        <IconChevronRight size={14} />
                                    </ActionIcon>
                                ),
                            },
                        ]}
                    />
                )}

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
