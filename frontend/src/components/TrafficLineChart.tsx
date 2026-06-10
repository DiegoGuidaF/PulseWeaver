import { useState } from "react";
import { Group, Paper, Skeleton, Text, UnstyledButton } from "@mantine/core";
import { LineChart } from "@mantine/charts";
import { IconChartLine } from "@tabler/icons-react";
import { formatChartLabel } from "@/lib/formatChartLabel";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import type { DashboardTrafficBucket } from "@/lib/api";

const SERIES: { name: string; color: string }[] = [
    { name: "Allowed", color: "teal.6" },
    { name: "Denied", color: "red.6" },
];

/** Converts a Mantine color token like "teal.6" to a CSS variable. */
function seriesColor(color: string) {
    const [palette, shade] = color.split(".");
    return `var(--mantine-color-${palette}-${shade})`;
}

interface TrafficLineChartProps {
    data: DashboardTrafficBucket[] | undefined;
    isLoading: boolean;
    timeRangeMs: number;
    h?: number;
    error?: unknown;
    onRetry?: () => void;
}

export function TrafficLineChart({ data, isLoading, timeRangeMs, h = 300, error, onRetry }: TrafficLineChartProps) {
    const [hiddenSeries, setHiddenSeries] = useState<Set<string>>(new Set());

    const chartData = (data ?? []).map((b) => ({
        timestamp: formatChartLabel(b.timestamp, timeRangeMs),
        Allowed: b.allow_count,
        Denied: b.deny_count,
    }));

    const visibleSeries = SERIES.filter((s) => !hiddenSeries.has(s.name));

    function toggle(name: string) {
        setHiddenSeries((prev) => {
            const next = new Set(prev);
            if (next.has(name)) next.delete(name);
            else next.add(name);
            return next;
        });
    }

    return (
        <Paper withBorder p="md" radius="md">
            <Group justify="space-between" mb="md">
                <Text fw={500}>Traffic over time</Text>
                <Group gap="sm">
                    {SERIES.map((s) => (
                        <UnstyledButton
                            key={s.name}
                            onClick={() => toggle(s.name)}
                            style={{
                                display: "flex",
                                alignItems: "center",
                                gap: 6,
                                minHeight: 24,
                                opacity: hiddenSeries.has(s.name) ? 0.35 : 1,
                            }}
                        >
                            <span
                                style={{
                                    display: "inline-block",
                                    width: 12,
                                    height: 12,
                                    borderRadius: 2,
                                    background: seriesColor(s.color),
                                    flexShrink: 0,
                                }}
                            />
                            <Text size="xs" c="dimmed">{s.name}</Text>
                        </UnstyledButton>
                    ))}
                </Group>
            </Group>
            {isLoading ? (
                <Skeleton h={h} />
            ) : error ? (
                <ErrorState error={error} title="Failed to load traffic" onRetry={onRetry} />
            ) : chartData.length === 0 ? (
                <EmptyState
                    icon={IconChartLine}
                    title="No traffic recorded yet"
                    description="Ensure PulseWeaver is configured as a forward-auth sidecar for your reverse proxy."
                />
            ) : (
                <LineChart
                    role="img"
                    aria-label="Traffic over time: line chart showing allowed and denied request counts"
                    h={h}
                    data={chartData}
                    dataKey="timestamp"
                    series={visibleSeries}
                    yAxisLabel="Requests"
                    yAxisProps={{ allowDecimals: false }}
                    curveType="monotone"
                    tooltipAnimationDuration={150}
                />
            )}
        </Paper>
    );
}
