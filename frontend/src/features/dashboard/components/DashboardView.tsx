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
import type { GetDashboardTrafficData } from "@/lib/api";

interface DashboardViewProps {
    from?: string;
    to?: string;
    timeRangeMs: number;
    granularity?: NonNullable<GetDashboardTrafficData["query"]>["granularity"];
}

export function DashboardView({ from, to, timeRangeMs, granularity }: DashboardViewProps) {
    const stats = useDashboardStats(from, to);
    const traffic = useDashboardTraffic(from, to, granularity);
    const services = useDashboardServices(from, to);
    const topDenied = useTopDeniedIPs(from, to);

    return (
        <Stack gap="lg">
            <DashboardStatCards data={stats.data} isLoading={stats.isLoading} />

            <SimpleGrid cols={{ base: 1, md: 2 }}>
                <TrafficLineChart data={traffic.data?.buckets} isLoading={traffic.isLoading} timeRangeMs={timeRangeMs} />
                <ServiceBarChart data={services.data?.services} isLoading={services.isLoading} />
            </SimpleGrid>

            <CountryStatsSection from={from} to={to} />

            <TopDeniedIPsTable data={topDenied.data?.ips} isLoading={topDenied.isLoading} />
        </Stack>
    );
}
