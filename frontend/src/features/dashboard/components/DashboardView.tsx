import { SimpleGrid, Stack } from "@mantine/core";
import { useDashboardStats } from "../hooks/useDashboardStats";
import { useDashboardTraffic } from "../hooks/useDashboardTraffic";
import { useDashboardServices } from "../hooks/useDashboardServices";
import { useTopDeniedIPs } from "../hooks/useTopDeniedIPs";
import { DashboardStatCards } from "./DashboardStatCards";
import { TrafficLineChart } from "@/components/TrafficLineChart";
import { ServiceBarChart } from "./ServiceBarChart";
import { TopDeniedIPsTable } from "./TopDeniedIPsTable";
import { CountryStatsSection } from "./CountryStatsSection";

interface DashboardViewProps {
    from?: string;
    to?: string;
    timeRangeMs: number;
}

export function DashboardView({ from, to, timeRangeMs }: DashboardViewProps) {
    const stats = useDashboardStats(from, to);
    const traffic = useDashboardTraffic(from, to);
    const services = useDashboardServices(from, to);
    const topDenied = useTopDeniedIPs(from, to);

    return (
        <Stack gap="lg">
            <DashboardStatCards
                data={stats.data}
                isLoading={stats.isLoading}
                error={stats.error}
                onRetry={() => stats.refetch()}
            />

            <SimpleGrid cols={{ base: 1, md: 2 }}>
                <TrafficLineChart
                    data={traffic.data?.buckets}
                    isLoading={traffic.isLoading}
                    timeRangeMs={timeRangeMs}
                    error={traffic.error}
                    onRetry={() => traffic.refetch()}
                />
                <ServiceBarChart
                    data={services.data?.services}
                    isLoading={services.isLoading}
                    error={services.error}
                    onRetry={() => services.refetch()}
                />
            </SimpleGrid>

            <CountryStatsSection from={from} to={to} />

            <TopDeniedIPsTable
                data={topDenied.data?.ips}
                isLoading={topDenied.isLoading}
                error={topDenied.error}
                onRetry={() => topDenied.refetch()}
            />
        </Stack>
    );
}
