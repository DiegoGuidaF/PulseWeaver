import { Card, Skeleton, Text } from "@mantine/core";
import { BarChart } from "@mantine/charts";
import { IconClockPause } from "@tabler/icons-react";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { formatChartLabel } from "@/lib/formatChartLabel";
import type { AddressHistoryBucket } from "@/lib/api";
import { RENEWAL_TIMING_BANDS, renewalTimingBandCounts } from "../constants";

interface AddressHistoryRiskChartProps {
    buckets: AddressHistoryBucket[] | undefined;
    timeRangeMs: number;
    isPending: boolean;
    error?: unknown;
    onRetry?: () => void;
}

/**
 * Distinct devices renewing per bucket, banded by how much of their lease TTL
 * the renewal consumed. The comfortable band is charted alongside the tight and
 * late ones on purpose: it is the denominator, and without it a handful of late
 * renewals fill the plot and read as a fleet-wide fault. The bucket series
 * arrives contiguous and zero-filled from the API, so quiet periods occupy
 * their real width instead of collapsing against the next busy one.
 */
export function AddressHistoryRiskChart({ buckets, timeRangeMs, isPending, error, onRetry }: AddressHistoryRiskChartProps) {
    const chartData = (buckets ?? []).map((b) => ({
        timestamp: formatChartLabel(b.timestamp, timeRangeMs),
        ...renewalTimingBandCounts(b),
    }));

    // An all-zero series now means "nothing renewed in this window" rather than
    // "nothing was at risk" — zero-filled buckets exist even where the fleet was
    // idle, so bucket count alone can no longer answer whether there is data.
    const hasAnyRenewal = (buckets ?? []).some(
        (b) =>
            b.ok_device_count + b.approaching_device_count + b.critical_device_count + b.breached_device_count > 0,
    );

    return (
        <Card withBorder padding="md">
            <Text fw={500}>Renewal timing vs. TTL</Text>
            <Text size="xs" c="dimmed" mb="sm">
                A renewal past its TTL means the address expired before the device next checked in — usually
                the device was asleep. Frequent ones mean the TTL is shorter than the device&apos;s real cadence.
            </Text>
            {isPending ? (
                <Skeleton height={200} radius="sm" />
            ) : error ? (
                <ErrorState error={error} title="Failed to load renewal timing" onRetry={onRetry} />
            ) : !hasAnyRenewal ? (
                <EmptyState
                    icon={IconClockPause}
                    title="No lease renewals in this window"
                    description="Nothing checked in over this period, so there is no renewal timing to compare against."
                />
            ) : (
                <BarChart
                    role="img"
                    aria-label="Renewal timing versus TTL: stacked bar chart of devices renewing per interval, banded by remaining TTL headroom"
                    h={220}
                    data={chartData}
                    dataKey="timestamp"
                    type="stacked"
                    series={RENEWAL_TIMING_BANDS.map((band) => ({ name: band.label, color: band.color }))}
                    yAxisLabel="Devices renewing"
                    yAxisProps={{ allowDecimals: false }}
                    // Zero-filled buckets make the series long enough that the
                    // default tick density collides labels at the right edge.
                    xAxisProps={{ minTickGap: 40 }}
                    tooltipAnimationDuration={150}
                    withLegend
                />
            )}
        </Card>
    );
}
