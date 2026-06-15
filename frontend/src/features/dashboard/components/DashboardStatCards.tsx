import { SimpleGrid, Paper, Text, Group, Skeleton } from "@mantine/core";
import {
    IconArrowsExchange,
    IconCheck,
    IconWorldQuestion,
    IconUserOff,
    IconUsers,
} from "@tabler/icons-react";
import { ErrorState } from "@/components/ErrorState";
import { InfoTooltip } from "@/components/InfoTooltip";
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
            info: "Every access decision PulseWeaver made in the selected time range.",
        },
        {
            label: "Allowed",
            value: (data?.allow_count ?? 0).toLocaleString(),
            subtitle: data ? pct(data.allow_count, data.total_requests) : null,
            icon: IconCheck,
            color: "teal",
            info: "Requests from a registered device with access to the host. The bulk of traffic should land here.",
        },
        {
            // ip_not_registered — denials from IPs with no registered device; internet noise.
            label: "Unknown IPs",
            value: (data?.deny_by_reason.ip_not_registered ?? 0).toLocaleString(),
            subtitle: data ? pct(data.deny_by_reason.ip_not_registered, data.total_requests) : null,
            icon: IconWorldQuestion,
            color: "gray",
            info: "Denied because the source IP has no registered device — usually internet background noise. Expected to be non-zero on an internet-facing host.",
        },
        {
            // host_not_allowed — a known IP denied a host it is not granted; a configured user blocked.
            label: "Blocked Users",
            value: (data?.deny_by_reason.host_not_allowed ?? 0).toLocaleString(),
            subtitle: data ? pct(data.deny_by_reason.host_not_allowed, data.total_requests) : null,
            icon: IconUserOff,
            color: "red",
            info: "A registered device denied a host it isn't granted. Ideally 0 — a non-zero value means a configured user is being locked out.",
        },
        {
            label: "Unique IPs",
            value: (data?.unique_ips ?? 0).toLocaleString(),
            subtitle: null,
            icon: IconUsers,
            color: "indigo",
            info: "Distinct source IP addresses seen in the selected time range.",
        },
    ];

    return (
        <SimpleGrid cols={{ base: 2, sm: 3, lg: 5 }}>
            {cards.map((card) => (
                <Paper key={card.label} withBorder p="md" radius="md">
                    <Group justify="space-between" mb="xs" wrap="nowrap">
                        <Group gap={4} align="center" wrap="nowrap">
                            <Text size="xs" c="dimmed" fw={500}>
                                {card.label}
                            </Text>
                            <InfoTooltip label={card.info} aria-label={`What "${card.label}" means`} />
                        </Group>
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
