import { useMemo } from "react";
import { Paper, Text, Table, Skeleton, Box, Group, Stack, Tooltip } from "@mantine/core";
import { IconGlobe } from "@tabler/icons-react";
import { countryFlagEmoji } from "@/lib/countryFlag";
import { EmptyState } from "@/components/EmptyState";
import type { AccessLogCountryStats } from "@/lib/api/types.gen";

interface TopCountriesTableProps {
    data: AccessLogCountryStats[] | undefined;
    isLoading: boolean;
    onCountryClick: (code: string) => void;
}

/** A legend dot + count line, reused for the allowed and denied rows of the bar tooltip. */
function TooltipRow({ color, count, label }: { color: string; count: number; label: string }) {
    return (
        <Group gap={6} wrap="nowrap">
            <Box w={8} h={8} style={{ borderRadius: "50%", backgroundColor: color }} />
            <Text size="xs">
                {count.toLocaleString()} {label}
            </Text>
        </Group>
    );
}

/** Two-segment bar showing the allowed (green) vs denied (red) breakdown of a country's traffic. */
function BreakdownBar({ allowed, denied }: { allowed: number; denied: number }) {
    const sum = allowed + denied;
    const allowedPct = sum > 0 ? (allowed / sum) * 100 : 0;
    return (
        <Tooltip
            withArrow
            label={
                <Stack gap={4}>
                    <TooltipRow color="var(--mantine-color-green-5)" count={allowed} label="allowed" />
                    <TooltipRow color="var(--mantine-color-red-5)" count={denied} label="denied" />
                </Stack>
            }
        >
            <Box
                aria-label={`${allowed.toLocaleString()} allowed, ${denied.toLocaleString()} denied`}
                style={{
                    display: "flex",
                    width: "100%",
                    minWidth: 56,
                    height: 8,
                    borderRadius: 4,
                    overflow: "hidden",
                    backgroundColor: "var(--mantine-color-red-6)",
                }}
            >
                <Box
                    style={{
                        width: `${allowedPct}%`,
                        backgroundColor: "var(--mantine-color-green-6)",
                    }}
                />
            </Box>
        </Tooltip>
    );
}

export function TopCountriesTable({
    data,
    isLoading,
    onCountryClick,
}: TopCountriesTableProps) {
    const sorted = useMemo(
        () => [...(data ?? [])].sort((a, b) => b.total - a.total).slice(0, 10),
        [data],
    );

    return (
        <Paper withBorder p="md" radius="md" h="100%">
            <Group justify="space-between" mb="md" wrap="nowrap">
                <Text fw={500}>Top Countries</Text>
                <Group gap="sm" wrap="nowrap">
                    <Text size="xs" c="green.6">
                        ● Allowed
                    </Text>
                    <Text size="xs" c="red.6">
                        ● Denied
                    </Text>
                </Group>
            </Group>
            {isLoading ? (
                <Skeleton h={200} />
            ) : sorted.length === 0 ? (
                <EmptyState
                    icon={IconGlobe}
                    title="No geographic data in this period"
                />
            ) : (
                <Table
                    striped
                    highlightOnHover
                    aria-label="Top countries by access requests"
                    style={{ tableLayout: "fixed", width: "100%" }}
                >
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th w={36}>#</Table.Th>
                            <Table.Th>Country</Table.Th>
                            <Table.Th w={110}>Breakdown</Table.Th>
                            <Table.Th w={72} style={{ textAlign: "right" }}>
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
                                    <Group gap={6} wrap="nowrap">
                                        <span>{countryFlagEmoji(row.country_code)}</span>
                                        <Tooltip
                                            label={row.country_name ?? row.country_code}
                                            withArrow
                                            openDelay={300}
                                        >
                                            <Text size="sm" truncate="end" style={{ flex: 1, minWidth: 0 }}>
                                                {row.country_name ?? row.country_code}
                                            </Text>
                                        </Tooltip>
                                    </Group>
                                </Table.Td>
                                <Table.Td style={{ width: "30%" }}>
                                    <BreakdownBar
                                        allowed={row.allowed}
                                        denied={row.denied}
                                    />
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
