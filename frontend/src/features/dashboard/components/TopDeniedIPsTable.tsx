import { Paper, Text, Table, Skeleton } from "@mantine/core";
import { IconShieldOff } from "@tabler/icons-react";
import { EmptyState } from "@/components/EmptyState";
import type { DashboardTopDeniedIp } from "@/lib/api";

interface TopDeniedIPsTableProps {
    data: DashboardTopDeniedIp[] | undefined;
    isLoading: boolean;
}

export function TopDeniedIPsTable({ data, isLoading }: TopDeniedIPsTableProps) {
    return (
        <Paper withBorder p="md" radius="md" maw={600}>
            <Text fw={500} mb="md">Top Denied IPs</Text>
            {isLoading ? (
                <Skeleton h={200} />
            ) : !data || data.length === 0 ? (
                <EmptyState
                    icon={IconShieldOff}
                    title="No denied requests in this period"
                />
            ) : (
                <Table striped highlightOnHover>
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>IP Address</Table.Th>
                            <Table.Th style={{ textAlign: "right" }}>Denied Count</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {data.map((row) => (
                            <Table.Tr key={row.ip}>
                                <Table.Td ff="monospace">{row.ip}</Table.Td>
                                <Table.Td style={{ textAlign: "right" }}>{row.count.toLocaleString()}</Table.Td>
                            </Table.Tr>
                        ))}
                    </Table.Tbody>
                </Table>
            )}
        </Paper>
    );
}
