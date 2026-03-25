import { ColorSwatch, Group, Paper, Text, Skeleton, Stack } from "@mantine/core";
import { DonutChart } from "@mantine/charts";
import type { DashboardServiceCount } from "@/lib/api";

interface ServiceDonutChartProps {
    data: DashboardServiceCount[] | undefined;
    isLoading: boolean;
}

const COLORS = [
    "blue.6", "teal.6", "violet.6", "orange.6", "pink.6",
    "cyan.6", "lime.6", "grape.6", "indigo.6", "yellow.6",
];

export function ServiceDonutChart({ data, isLoading }: ServiceDonutChartProps) {
    const chartData = (data ?? []).map((s, i) => ({
        name: s.host || "(unknown)",
        value: s.allow_count + s.deny_count,
        color: COLORS[i % COLORS.length],
    }));

    return (
        <Paper withBorder p="md" radius="md">
            <Text fw={500} mb="md">Requests by Service</Text>
            {isLoading ? (
                <Skeleton h={300} />
            ) : chartData.length === 0 ? (
                <Text c="dimmed" ta="center" py="xl">No service data for this period.</Text>
            ) : (
                <Stack gap="sm">
                    <DonutChart
                        h={260}
                        data={chartData}
                        withTooltip
                        tooltipDataSource="segment"
                        valueFormatter={(v) => `${new Intl.NumberFormat().format(v)} reqs`}
                        size={200}
                        thickness={18}
                        paddingAngle={chartData.length > 1 ? 3 : 0}
                        chartLabel={`${chartData.reduce((sum, d) => sum + d.value, 0)}`}
                    />
                    <Group gap="md" justify="center" wrap="wrap">
                        {chartData.map((entry) => (
                            <Group key={entry.name} gap={6}>
                                <ColorSwatch color={`var(--mantine-color-${entry.color.replace(".", "-")})`} size={12} withShadow={false} />
                                <Text size="xs">{entry.name}</Text>
                            </Group>
                        ))}
                    </Group>
                </Stack>
            )}
        </Paper>
    );
}
