import { useMemo } from "react";
import { Paper, Text, Table, Skeleton } from "@mantine/core";
import { IconGlobe } from "@tabler/icons-react";
import { countryFlagEmoji } from "@/lib/countryFlag";
import { EmptyState } from "@/components/EmptyState";
import type { AccessLogCountryStats } from "@/lib/api/types.gen";

type Metric = "denied" | "total";

interface TopCountriesTableProps {
    data: AccessLogCountryStats[] | undefined;
    isLoading: boolean;
    metric: Metric;
    onCountryClick: (code: string) => void;
}

export function TopCountriesTable({
    data,
    isLoading,
    metric,
    onCountryClick,
}: TopCountriesTableProps) {
    const sorted = useMemo(
        () =>
            [...(data ?? [])]
                .sort((a, b) => b[metric] - a[metric])
                .slice(0, 10),
        [data, metric],
    );

    return (
        <Paper withBorder p="md" radius="md" h="100%">
            <Text fw={500} mb="md">
                Top Countries
            </Text>
            {isLoading ? (
                <Skeleton h={200} />
            ) : sorted.length === 0 ? (
                <EmptyState
                    icon={IconGlobe}
                    title="No geographic data in this period"
                />
            ) : (
                <Table striped highlightOnHover aria-label="Top countries by access requests">
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>#</Table.Th>
                            <Table.Th>Country</Table.Th>
                            <Table.Th style={{ textAlign: "right" }}>
                                Allowed
                            </Table.Th>
                            <Table.Th style={{ textAlign: "right" }}>
                                Denied
                            </Table.Th>
                            <Table.Th style={{ textAlign: "right" }}>
                                Total
                            </Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {sorted.map((row, i) => (
                            <Table.Tr
                                key={row.country_code}
                                style={{ cursor: "pointer" }}
                                onClick={() => onCountryClick(row.country_code)}
                            >
                                <Table.Td c="dimmed">{i + 1}</Table.Td>
                                <Table.Td>
                                    {countryFlagEmoji(row.country_code)}{" "}
                                    {row.country_name ?? row.country_code}
                                </Table.Td>
                                <Table.Td style={{ textAlign: "right" }}>
                                    {row.allowed.toLocaleString()}
                                </Table.Td>
                                <Table.Td style={{ textAlign: "right" }}>
                                    {row.denied.toLocaleString()}
                                </Table.Td>
                                <Table.Td style={{ textAlign: "right" }}>
                                    {row.total.toLocaleString()}
                                </Table.Td>
                            </Table.Tr>
                        ))}
                    </Table.Tbody>
                </Table>
            )}
        </Paper>
    );
}
