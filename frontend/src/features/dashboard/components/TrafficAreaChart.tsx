import { Paper, Text, Skeleton } from "@mantine/core";
import { AreaChart } from "@mantine/charts";
import dayjs from "dayjs";
import type { DashboardTrafficBucket } from "@/lib/api";

interface TrafficAreaChartProps {
    data: DashboardTrafficBucket[] | undefined;
    isLoading: boolean;
}

function formatBucketLabel(ts: string): string {
    const d = dayjs(ts);
    // If the timestamp is midnight, show the date; otherwise show time
    return d.hour() === 0 && d.minute() === 0
        ? d.format("MMM D")
        : d.format("HH:mm");
}

export function TrafficAreaChart({ data, isLoading }: TrafficAreaChartProps) {
    const chartData = (data ?? []).map((b) => ({
        timestamp: formatBucketLabel(b.timestamp),
        Allowed: b.allow_count,
        Denied: b.deny_count,
    }));

    return (
        <Paper withBorder p="md" radius="md">
            <Text fw={500} mb="md">Traffic Over Time</Text>
            {isLoading ? (
                <Skeleton h={300} />
            ) : chartData.length === 0 ? (
                <Text c="dimmed" ta="center" py="xl">No traffic data for this period.</Text>
            ) : (
                <AreaChart
                    h={300}
                    data={chartData}
                    dataKey="timestamp"
                    series={[
                        { name: "Allowed", color: "teal.6" },
                        { name: "Denied", color: "red.6" },
                    ]}
                />
            )}
        </Paper>
    );
}
