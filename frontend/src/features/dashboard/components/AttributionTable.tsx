import { useMemo, useState } from "react";
import { Paper, Text, Table, Group, Skeleton, ScrollArea, Anchor } from "@mantine/core";
import { IconChartBar } from "@tabler/icons-react";
import { useNavigate } from "react-router-dom";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import type { DashboardAttributionCount } from "@/lib/api";

interface AttributionTableProps {
    title: string;
    /** Column header for the entity name (e.g. "Policy", "User", "Device"). */
    entityHeader: string;
    /** Plural of entityHeader, used in the row-count label. */
    entityHeaderPlural: string;
    data: DashboardAttributionCount[] | undefined;
    isLoading: boolean;
    error?: unknown;
    onRetry?: () => void;
    emptyText: string;
    /** Destination for a row, or undefined when the entity has no detail page (e.g. deleted, or kind not linkable). */
    rowHref?: (row: DashboardAttributionCount) => string | undefined;
}

/** Rows shown before the operator expands the long tail. Sized so three tables sit side by side. */
const COLLAPSED_ROWS = 8;

function total(row: DashboardAttributionCount): number {
    return row.allow_count + row.deny_count;
}

export function AttributionTable({
    title,
    entityHeader,
    entityHeaderPlural,
    data,
    isLoading,
    error,
    onRetry,
    emptyText,
    rowHref,
}: AttributionTableProps) {
    const [expanded, setExpanded] = useState(false);
    const navigate = useNavigate();

    const sorted = useMemo(
        () => (data ? [...data].sort((a, b) => total(b) - total(a)) : []),
        [data],
    );

    const hasMore = sorted.length > COLLAPSED_ROWS;
    const visible = expanded ? sorted : sorted.slice(0, COLLAPSED_ROWS);

    return (
        <Paper withBorder p="md" radius="md">
            <Group justify="space-between" mb="md" wrap="nowrap">
                <Text fw={500}>{title}</Text>
                {sorted.length > 0 && (
                    <Text size="xs" c="dimmed">
                        {sorted.length.toLocaleString()}{" "}
                        {sorted.length === 1 ? entityHeader.toLowerCase() : entityHeaderPlural.toLowerCase()}
                    </Text>
                )}
            </Group>

            {isLoading ? (
                <Skeleton h={180} />
            ) : error ? (
                <ErrorState error={error} title={`Failed to load ${entityHeader.toLowerCase()} traffic`} onRetry={onRetry} />
            ) : sorted.length === 0 ? (
                <EmptyState icon={IconChartBar} title={emptyText} />
            ) : (
                <>
                    <ScrollArea.Autosize mah={expanded ? 360 : undefined} type="auto">
                        <Table highlightOnHover aria-label={title} stickyHeader={expanded}>
                            <Table.Thead>
                                <Table.Tr>
                                    <Table.Th>{entityHeader}</Table.Th>
                                    <Table.Th style={{ textAlign: "right" }}>Allowed</Table.Th>
                                    <Table.Th style={{ textAlign: "right" }}>Denied</Table.Th>
                                    <Table.Th style={{ textAlign: "right" }}>Total</Table.Th>
                                </Table.Tr>
                            </Table.Thead>
                            <Table.Tbody>
                                {visible.map((row) => {
                                    const href = rowHref?.(row);
                                    return (
                                    <Table.Tr
                                        key={row.entity_id ?? `deleted:${row.entity_name}`}
                                        style={href ? { cursor: "pointer" } : undefined}
                                        onClick={href ? () => navigate(href) : undefined}
                                    >
                                        <Table.Td>
                                            <Text size="sm" truncate="end" maw={180} title={row.entity_name}>
                                                {row.entity_name}
                                                {row.entity_id == null && (
                                                    <Text span size="xs" c="dimmed">
                                                        {" "}
                                                        (deleted)
                                                    </Text>
                                                )}
                                            </Text>
                                        </Table.Td>
                                        <Table.Td style={{ textAlign: "right" }}>
                                            <Text size="sm" c={row.allow_count > 0 ? "teal" : "dimmed"}>
                                                {row.allow_count.toLocaleString()}
                                            </Text>
                                        </Table.Td>
                                        <Table.Td style={{ textAlign: "right" }}>
                                            <Text size="sm" c={row.deny_count > 0 ? "red" : "dimmed"}>
                                                {row.deny_count.toLocaleString()}
                                            </Text>
                                        </Table.Td>
                                        <Table.Td style={{ textAlign: "right" }} fw={500}>
                                            {total(row).toLocaleString()}
                                        </Table.Td>
                                    </Table.Tr>
                                    );
                                })}
                            </Table.Tbody>
                        </Table>
                    </ScrollArea.Autosize>

                    {hasMore && (
                        <Anchor
                            component="button"
                            type="button"
                            size="sm"
                            mt="sm"
                            onClick={() => setExpanded((v) => !v)}
                        >
                            {expanded ? `Show top ${COLLAPSED_ROWS}` : `Show all ${sorted.length.toLocaleString()}`}
                        </Anchor>
                    )}
                </>
            )}
        </Paper>
    );
}
