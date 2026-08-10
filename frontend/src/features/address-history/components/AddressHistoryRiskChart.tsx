import { Card, Skeleton, Text } from "@mantine/core";
import { BarChart } from "@mantine/charts";
import { IconShieldCheck } from "@tabler/icons-react";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { formatChartLabel } from "@/lib/formatChartLabel";
import { TtlRisk, type AddressHistoryBucket } from "@/lib/api";
import { RISK_BAND_CHART_COLORS, TTL_RISK_LABELS } from "../constants";

const RISK_BANDS = [TtlRisk.APPROACHING, TtlRisk.CRITICAL, TtlRisk.BREACHED] as const;

interface AddressHistoryRiskChartProps {
    buckets: AddressHistoryBucket[] | undefined;
    timeRangeMs: number;
    isPending: boolean;
    error?: unknown;
    onRetry?: () => void;
}

/**
 * Stacked bar chart of distinct at-risk devices per bucket. The measure is
 * devices, not events, so a fleet with no device at risk is a legitimate and
 * reassuring answer ("fleet healthy") rather than a missing-data state.
 */
export function AddressHistoryRiskChart({ buckets, timeRangeMs, isPending, error, onRetry }: AddressHistoryRiskChartProps) {
    const chartData = (buckets ?? []).map((b) => ({
        timestamp: formatChartLabel(b.timestamp, timeRangeMs),
        [TTL_RISK_LABELS[TtlRisk.APPROACHING]]: b.approaching_device_count,
        [TTL_RISK_LABELS[TtlRisk.CRITICAL]]: b.critical_device_count,
        [TTL_RISK_LABELS[TtlRisk.BREACHED]]: b.breached_device_count,
    }));

    // The bucket list is the set of buckets carrying any matching event, not
    // any at-risk device — a window of purely routine traffic still returns
    // buckets, with all three band counts at zero. Emptiness therefore has to
    // be measured on the bands themselves, or a healthy fleet renders an
    // all-zero stacked chart instead of the reassurance it has earned.
    const hasAnyAtRiskDevice = (buckets ?? []).some(
        (b) => b.approaching_device_count + b.critical_device_count + b.breached_device_count > 0,
    );

    return (
        <Card withBorder padding="md">
            <Text fw={500} mb="sm">Devices at risk over time</Text>
            {isPending ? (
                <Skeleton height={200} radius="sm" />
            ) : error ? (
                <ErrorState error={error} title="Failed to load risk history" onRetry={onRetry} />
            ) : !hasAnyAtRiskDevice ? (
                <EmptyState
                    icon={IconShieldCheck}
                    color="green"
                    title="No devices at risk in this period"
                    description="Every device's lease renewed comfortably within its TTL."
                />
            ) : (
                <BarChart
                    role="img"
                    aria-label="Devices at risk over time: stacked bar chart of approaching, critical, and breached device counts"
                    h={200}
                    data={chartData}
                    dataKey="timestamp"
                    type="stacked"
                    series={RISK_BANDS.map((risk) => ({
                        name: TTL_RISK_LABELS[risk],
                        color: RISK_BAND_CHART_COLORS[risk],
                    }))}
                    yAxisLabel="Devices at risk"
                    yAxisProps={{ allowDecimals: false }}
                    tooltipAnimationDuration={150}
                    withLegend
                />
            )}
        </Card>
    );
}
