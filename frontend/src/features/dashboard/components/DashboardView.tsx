import { Stack, SimpleGrid } from "@mantine/core";
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
            <DashboardStatCards data={stats.data} isLoading={stats.isLoading} />

            <SimpleGrid cols={{ base: 1, md: 2 }}>
                <TrafficLineChart data={traffic.data?.buckets} isLoading={traffic.isLoading} timeRangeMs={timeRangeMs} />
                <ServiceBarChart data={services.data?.services} isLoading={services.isLoading} />
            </SimpleGrid>

            <TopDeniedIPsTable data={topDenied.data?.ips} isLoading={topDenied.isLoading} />

            <CountryStatsSection from={from} to={to} />
        </Stack>
    );
}
