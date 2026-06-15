import { SimpleGrid, Paper, Text, Group, Stack, Skeleton, ThemeIcon } from "@mantine/core";
import {
    IconShieldHalf,
    IconLock,
    IconRouteOff,
    IconArrowsShuffle,
    IconBellRinging,
    type IconProps,
} from "@tabler/icons-react";
import { Link } from "react-router-dom";
import type { ComponentType } from "react";
import { ROUTES } from "@/lib/routes";
import { ErrorState } from "@/components/ErrorState";
import { InfoTooltip } from "@/components/InfoTooltip";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import type { DashboardPosture } from "@/lib/api";

interface PostureStripProps {
    data: DashboardPosture | undefined;
    isLoading: boolean;
    error?: unknown;
    onRetry?: () => void;
}

interface PostureCardSpec {
    label: string;
    value: number;
    hint: string;
    /** Tooltip text: what the signal is and what a healthy value looks like. */
    info: string;
    icon: ComponentType<IconProps>;
    to: string;
    /** Draw attention (red) only when the count is non-zero — a true exposure/lockout signal. */
    alertWhenSet?: boolean;
}

function PostureCard({ spec, isLoading }: { spec: PostureCardSpec; isLoading: boolean }) {
    const alert = spec.alertWhenSet === true && spec.value > 0;
    return (
        <Paper
            component={Link}
            to={spec.to}
            withBorder
            p="md"
            radius="md"
            style={{
                textDecoration: "none",
                color: "inherit",
                borderColor: alert ? "var(--mantine-color-red-6)" : undefined,
            }}
        >
            <Group justify="space-between" mb="xs" wrap="nowrap">
                <Group gap={4} align="center" wrap="nowrap">
                    <Text size="xs" c="dimmed" fw={500}>
                        {spec.label}
                    </Text>
                    <InfoTooltip label={spec.info} aria-label={`What "${spec.label}" means`} />
                </Group>
                <ThemeIcon variant="light" color={alert ? "red" : "indigo"} size="md" radius="md">
                    <spec.icon size={16} stroke={1.5} />
                </ThemeIcon>
            </Group>
            {isLoading ? (
                <Skeleton h={28} w="50%" />
            ) : (
                <Text fw={700} fz="xl" c={alert ? "red" : undefined}>
                    {spec.value.toLocaleString()}
                </Text>
            )}
            <Text size="xs" c="dimmed" mt={2}>
                {spec.hint}
            </Text>
        </Paper>
    );
}

export function PostureStrip({ data, isLoading, error, onRetry }: PostureStripProps) {
    const formatDateTime = useDateFormatter();

    if (error) {
        return <ErrorState error={error} title="Failed to load posture" onRetry={onRetry} />;
    }

    const cards: PostureCardSpec[] = [
        {
            label: "Bypass users",
            value: data?.users.bypass ?? 0,
            hint: "reach every host",
            info: "Users whose devices reach every host, skipping the per-host allow-list. Keep this low — each one is a broad standing grant.",
            icon: IconShieldHalf,
            to: ROUTES.accessUsers,
        },
        {
            label: "Locked-out users",
            value: data?.users.live_no_host_access ?? 0,
            hint: "live, every request denied",
            info: "Users with a live device whose every request is currently denied. Should be 0 — a non-zero value usually means a misconfigured grant.",
            icon: IconLock,
            to: ROUTES.policyAudit,
            alertWhenSet: true,
        },
        {
            label: "Bypass-check policies",
            value: data?.network_policies.bypass_host_check ?? 0,
            hint: `of ${(data?.network_policies.enabled ?? 0).toLocaleString()} enabled`,
            info: "Enabled network policies that skip the per-host allow-list. Keep this low — they widen exposure for everyone they match.",
            icon: IconRouteOff,
            to: ROUTES.accessNetworkPolicies,
        },
        {
            label: "Shared IPs",
            value: data?.shared_ip_count ?? 0,
            hint: "claimed by multiple users",
            info: "IP addresses currently claimed by more than one user. Usually 0; a non-zero value blurs per-user traffic attribution.",
            icon: IconArrowsShuffle,
            to: ROUTES.policyAudit,
        },
    ];

    const pendingSuggestions = data?.pending_suggestion_count ?? 0;

    return (
        <Stack gap="xs">
            <Group justify="space-between" align="baseline" wrap="wrap">
                <Text fw={600}>Security posture</Text>
                {data && (
                    <Group gap={4} align="center" wrap="nowrap">
                        <Text size="xs" c="dimmed">
                            Current state · cache as of {formatDateTime(data.refreshed_at)}
                        </Text>
                        <InfoTooltip
                            label="The posture cache only recomputes when a device address or access setting changes. An older timestamp just means nothing has changed since — these figures are still current."
                            aria-label="What the cache timestamp means"
                        />
                    </Group>
                )}
            </Group>

            <SimpleGrid cols={{ base: 2, sm: 2, lg: 4 }}>
                {cards.map((spec) => (
                    <PostureCard key={spec.label} spec={spec} isLoading={isLoading} />
                ))}
            </SimpleGrid>

            {/* Pending suggestions is a live read, fresher than the cache snapshot above —
                kept on its own row so the "as of" label does not appear to cover it. */}
            {pendingSuggestions > 0 && (
                <Paper
                    component={Link}
                    to={ROUTES.accessHosts}
                    withBorder
                    p="sm"
                    radius="md"
                    style={{ textDecoration: "none", color: "inherit" }}
                >
                    <Group gap="sm" wrap="nowrap">
                        <ThemeIcon variant="light" color="indigo" size="md" radius="md">
                            <IconBellRinging size={16} stroke={1.5} />
                        </ThemeIcon>
                        <Text size="sm">
                            <Text span fw={700}>
                                {pendingSuggestions.toLocaleString()}
                            </Text>{" "}
                            pending host {pendingSuggestions === 1 ? "suggestion" : "suggestions"} to review
                        </Text>
                    </Group>
                </Paper>
            )}
        </Stack>
    );
}
