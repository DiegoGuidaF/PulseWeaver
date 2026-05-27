import { Group, Paper, Text, Skeleton, Progress, Stack } from "@mantine/core";
import { IconWorldOff } from "@tabler/icons-react";
import { EmptyState } from "@/components/EmptyState";
import type { DashboardServiceCount } from "@/lib/api";

interface ServiceBarChartProps {
    data: DashboardServiceCount[] | undefined;
    isLoading: boolean;
}

const COLORS = [
    "blue", "teal", "violet", "orange", "pink",
    "cyan", "lime", "grape", "indigo", "yellow",
];

export function ServiceBarChart({ data, isLoading }: ServiceBarChartProps) {
    const entries = (data ?? [])
        .map((s, i) => ({
            name: s.host || "(unknown)",
            value: s.allow_count + s.deny_count,
            color: COLORS[i % COLORS.length],
        }))
        .sort((a, b) => b.value - a.value);

    const total = entries.reduce((sum, e) => sum + e.value, 0);
    const max = entries[0]?.value ?? 1;

    return (
        <Paper withBorder p="md" radius="md">
            <Group justify="space-between" mb="md">
                <Text fw={500}>Requests by Service</Text>
                {!isLoading && entries.length > 0 && (
                    <Text size="sm" c="dimmed">
                        {total.toLocaleString()} total
                    </Text>
                )}
            </Group>
            {isLoading ? (
                <Skeleton h={300} />
            ) : entries.length === 0 ? (
                <EmptyState
                    icon={IconWorldOff}
                    title="No service data for this period"
                />
            ) : (
                <Stack gap="sm">
                    {entries.map((entry) => (
                        <Group key={entry.name} gap="sm" wrap="nowrap">
                            <Text size="sm" w={140} truncate="end" title={entry.name}>
                                {entry.name}
                            </Text>
                            <Progress.Root size="lg" radius="sm" style={{ flex: 1 }}>
                                <Progress.Section
                                    value={(entry.value / max) * 100}
                                    color={entry.color}
                                    aria-label={`${entry.name}: ${entry.value.toLocaleString()} requests`}
                                />
                            </Progress.Root>
                            <Text size="sm" ff="monospace" w={70} ta="right">
                                {entry.value.toLocaleString()}
                            </Text>
                        </Group>
                    ))}
                </Stack>
            )}
        </Paper>
    );
}
