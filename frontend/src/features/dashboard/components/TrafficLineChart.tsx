import { Paper, Text, Skeleton } from "@mantine/core";
import { LineChart } from "@mantine/charts";
import { IconChartLine } from "@tabler/icons-react";
import { formatChartLabel } from "@/lib/formatChartLabel";
import { EmptyState } from "@/components/EmptyState";
import type { DashboardTrafficBucket } from "@/lib/api";

interface TrafficLineChartProps {
    data: DashboardTrafficBucket[] | undefined;
    isLoading: boolean;
    timeRangeMs: number;
}

export function TrafficLineChart({ data, isLoading, timeRangeMs }: TrafficLineChartProps) {
    const chartData = (data ?? []).map((b) => ({
        timestamp: formatChartLabel(b.timestamp, timeRangeMs),
        Allowed: b.allow_count,
        Denied: b.deny_count,
    }));

    return (
        <Paper withBorder p="md" radius="md">
            <Text fw={500} mb="md">Traffic Over Time</Text>
            {isLoading ? (
                <Skeleton h={300} />
            ) : chartData.length === 0 ? (
                <EmptyState
                    icon={IconChartLine}
                    title="No traffic recorded yet"
                    description="Ensure PulseWeaver is configured as a forward-auth sidecar for your reverse proxy."
                />
            ) : (
                <LineChart
                    h={300}
                    data={chartData}
                    dataKey="timestamp"
                    series={[
                        { name: "Allowed", color: "teal.6" },
                        { name: "Denied", color: "red.6" },
                    ]}
                    yAxisLabel="Requests"
                    curveType="monotone"
                    tooltipAnimationDuration={150}
                />
            )}
        </Paper>
    );
}
