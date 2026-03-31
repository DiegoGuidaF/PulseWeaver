import { SimpleGrid, Paper, Text, Group, Skeleton } from "@mantine/core";
import {
    IconArrowsExchange,
    IconCheck,
    IconX,
    IconUsers,
} from "@tabler/icons-react";
import type { DashboardStats } from "@/lib/api";

interface DashboardStatCardsProps {
    data: DashboardStats | undefined;
    isLoading: boolean;
}

function pct(count: number, total: number): string {
    if (total === 0) return "0%";
    return `${((count / total) * 100).toFixed(1)}%`;
}

export function DashboardStatCards({ data, isLoading }: DashboardStatCardsProps) {
    const cards = [
        {
            label: "Total Requests",
            value: data?.total_requests ?? 0,
            subtitle: null,
            icon: IconArrowsExchange,
            color: "indigo",
        },
        {
            label: "Allowed",
            value: data?.allowed_count ?? 0,
            subtitle: data ? pct(data.allowed_count, data.total_requests) : null,
            icon: IconCheck,
            color: "teal",
        },
        {
            label: "Denied",
            value: data?.denied_count ?? 0,
            subtitle: data ? pct(data.denied_count, data.total_requests) : null,
            icon: IconX,
            color: "red",
        },
        {
            label: "Unique IPs",
            value: data?.unique_ips ?? 0,
            subtitle: null,
            icon: IconUsers,
            color: "indigo",
        },
    ];

    return (
        <SimpleGrid cols={{ base: 2, sm: 4 }}>
            {cards.map((card) => (
                <Paper key={card.label} withBorder p="md" radius="md">
                    <Group justify="space-between" mb="xs">
                        <Text size="xs" c="dimmed" fw={500} tt="uppercase">
                            {card.label}
                        </Text>
                        <card.icon size={20} color={`var(--mantine-color-${card.color}-6)`} stroke={1.5} />
                    </Group>
                    {isLoading ? (
                        <Skeleton h={28} w="60%" />
                    ) : (
                        <Group align="baseline" gap="xs">
                            <Text fw={700} fz="xl">
                                {card.value.toLocaleString()}
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
