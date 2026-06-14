import { SimpleGrid, Paper, Text, Group, Skeleton } from "@mantine/core";
import {
    IconArrowsExchange,
    IconCheck,
    IconWorldQuestion,
    IconUserOff,
    IconUsers,
} from "@tabler/icons-react";
import { ErrorState } from "@/components/ErrorState";
import type { DashboardStats } from "@/lib/api";

interface DashboardStatCardsProps {
    data: DashboardStats | undefined;
    isLoading: boolean;
    error?: unknown;
    onRetry?: () => void;
}

function pct(count: number, total: number): string {
    if (total === 0) return "0%";
    return `${((count / total) * 100).toFixed(1)}%`;
}

export function DashboardStatCards({ data, isLoading, error, onRetry }: DashboardStatCardsProps) {
    if (error) {
        return <ErrorState error={error} title="Failed to load stats" onRetry={onRetry} />;
    }

    const cards = [
        {
            label: "Total Requests",
            value: (data?.total_requests ?? 0).toLocaleString(),
            subtitle: null,
            icon: IconArrowsExchange,
            color: "indigo",
        },
        {
            label: "Allowed",
            value: (data?.allow_count ?? 0).toLocaleString(),
            subtitle: data ? pct(data.allow_count, data.total_requests) : null,
            icon: IconCheck,
            color: "teal",
        },
        {
            // ip_not_registered — denials from IPs with no registered device; internet noise.
            label: "Unknown IPs",
            value: (data?.deny_by_reason.ip_not_registered ?? 0).toLocaleString(),
            subtitle: data ? pct(data.deny_by_reason.ip_not_registered, data.total_requests) : null,
            icon: IconWorldQuestion,
            color: "gray",
        },
        {
            // host_not_allowed — a known IP denied a host it is not granted; a configured user blocked.
            label: "Blocked Users",
            value: (data?.deny_by_reason.host_not_allowed ?? 0).toLocaleString(),
            subtitle: data ? pct(data.deny_by_reason.host_not_allowed, data.total_requests) : null,
            icon: IconUserOff,
            color: "red",
        },
        {
            label: "Unique IPs",
            value: (data?.unique_ips ?? 0).toLocaleString(),
            subtitle: null,
            icon: IconUsers,
            color: "indigo",
        },
    ];

    return (
        <SimpleGrid cols={{ base: 2, sm: 3, lg: 5 }}>
            {cards.map((card) => (
                <Paper key={card.label} withBorder p="md" radius="md">
                    <Group justify="space-between" mb="xs">
                        <Text size="xs" c="dimmed" fw={500}>
                            {card.label}
                        </Text>
                        <card.icon size={20} color={`var(--mantine-color-${card.color}-6)`} stroke={1.5} />
                    </Group>
                    {isLoading ? (
                        <Skeleton h={28} w="60%" />
                    ) : (
                        <Group align="baseline" gap="xs">
                            <Text fw={700} fz="xl">
                                {card.value}
                            </Text>
                            {card.subtitle && (
                                <Text size="sm" c="dimmed">
                                    {card.subtitle}
                                </Text>
                            )}
                        </Group>
                    )}
                </Paper>
            ))}
        </SimpleGrid>
    );
}
