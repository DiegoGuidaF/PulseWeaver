import { Paper, Text, Table, Skeleton } from "@mantine/core";
import { IconShieldOff } from "@tabler/icons-react";
import { useNavigate } from "react-router-dom";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { GeoCell } from "@/components/GeoCell";
import { ROUTES } from "@/lib/routes";
import type { DashboardTopDeniedIp } from "@/lib/api";

interface TopDeniedIPsTableProps {
    data: DashboardTopDeniedIp[] | undefined;
    isLoading: boolean;
    error?: unknown;
    onRetry?: () => void;
}

export function TopDeniedIPsTable({ data, isLoading, error, onRetry }: TopDeniedIPsTableProps) {
    const navigate = useNavigate();
    return (
        <Paper withBorder p="md" radius="md">
            <Text fw={500} mb="md">Top Denied IPs</Text>
            {isLoading ? (
                <Skeleton h={200} />
            ) : error ? (
                <ErrorState error={error} title="Failed to load denied IPs" onRetry={onRetry} />
            ) : !data || data.length === 0 ? (
                <EmptyState
                    icon={IconShieldOff}
                    title="No denied requests in this period"
                />
            ) : (
                <Table striped highlightOnHover aria-label="Top denied IP addresses">
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>IP Address</Table.Th>
                            <Table.Th>Location</Table.Th>
                            <Table.Th style={{ textAlign: "right" }}>Denied Count</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {data.map((row) => (
                            <Table.Tr
                                key={row.ip}
                                style={{ cursor: "pointer" }}
                                onClick={() => navigate(`${ROUTES.accessLog}?client_ip=${encodeURIComponent(row.ip)}`)}
                            >
                                <Table.Td ff="monospace">{row.ip}</Table.Td>
                                <Table.Td>
                                    <GeoCell geo={row.geo} size="sm" />
                                </Table.Td>
                                <Table.Td style={{ textAlign: "right" }}>{row.count.toLocaleString()}</Table.Td>
                            </Table.Tr>
                        ))}
                    </Table.Tbody>
                </Table>
            )}
        </Paper>
    );
}
