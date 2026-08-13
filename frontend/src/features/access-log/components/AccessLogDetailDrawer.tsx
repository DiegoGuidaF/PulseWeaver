import { ActionIcon, Badge, Code, Divider, Drawer, Group, Skeleton, Stack, Text, Title, Tooltip } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { useMemo, useState } from "react";
import type { AccessLogDetail } from "@/lib/api";
import { POLICY_DENY_REASON_LABELS } from "@/lib/policyDenyReasons";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { countryFlagEmoji } from "@/lib/countryFlag";
import { useClipboard } from "@/hooks/useClipboard";
import { IconCopy, IconTrashOff } from "@tabler/icons-react";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { isEntryUnavailable, useAccessLogEntry } from "../hooks/useAccessLogEntry";

interface AccessLogDetailDrawerProps {
    requestId: number | null;
    opened: boolean;
    onClose: () => void;
}

export function AccessLogDetailDrawer({
    requestId,
    opened,
    onClose,
}: AccessLogDetailDrawerProps) {
    // A fixed-width "lg" drawer is wider than a phone viewport and leaves its
    // header clipped above the visible area; fill the screen below the tablet
    // breakpoint so the title and close button stay reachable.
    const isMobile = useMediaQuery("(max-width: 48em)", false, { getInitialValueInEffect: false });

    // The URL param clears the moment the drawer starts closing, so keep querying
    // the last id through the close animation — otherwise the body blanks mid-slide.
    const [lastId, setLastId] = useState(requestId);
    if (requestId != null && requestId !== lastId) setLastId(requestId);

    const { data: detail, isPending, error, refetch } = useAccessLogEntry(lastId);

    return (
        <Drawer
            opened={opened}
            onClose={onClose}
            position="right"
            size={isMobile ? "100%" : "lg"}
            title={<Title order={2} size="h4">Request Detail</Title>}
        >
            {isPending ? (
                <Stack gap="md">
                    {Array.from({ length: 6 }).map((_, i) => (
                        <Skeleton key={i} height={28} radius="sm" />
                    ))}
                </Stack>
            ) : isEntryUnavailable(error) ? (
                <EmptyState
                    icon={IconTrashOff}
                    title="This request is no longer available"
                    description="Access-log entries are removed once they pass the retention window, so an older link can outlive the request it points to."
                />
            ) : error ? (
                <ErrorState error={error} onRetry={() => refetch()} />
            ) : (
                detail && <DetailBody detail={detail} />
            )}
        </Drawer>
    );
}

function DetailBody({ detail }: { detail: AccessLogDetail }) {
    const formatDateTime = useDateFormatter();
    const { copy } = useClipboard();
    const headersJson = useMemo(
        () => (Object.keys(detail.headers).length > 0 ? JSON.stringify(detail.headers, null, 2) : null),
        [detail.headers],
    );

    return (
        <Stack gap="md">
            <Group gap="xs">
                <Badge color={detail.outcome ? "green" : "red"} size="lg">
                    {detail.outcome ? "Allowed" : "Denied"}
                </Badge>
                {!detail.outcome && detail.deny_reason && (
                    <Badge color="gray" variant="light" size="lg">
                        {POLICY_DENY_REASON_LABELS[detail.deny_reason]}
                    </Badge>
                )}
            </Group>

            <Stack gap="xs">
                <Title order={3} size="h5">Request</Title>
                <Divider />
                <LabelValue label="Time" value={formatDateTime(detail.created_at)} />
                <LabelValue label="Client IP" value={detail.client_ip} mono />
                {detail.xff_chain && (
                    <LabelValue label="XFF Chain" value={detail.xff_chain} mono />
                )}
                {detail.http_method && (
                    <LabelValue label="Method" value={detail.http_method} />
                )}
                {detail.target_host && (
                    <LabelValue label="Host" value={detail.target_host} mono />
                )}
                {detail.target_uri && (
                    <LabelValue label="URI" value={detail.target_uri} mono />
                )}
            </Stack>

            <Stack gap="xs">
                <Title order={3} size="h5">Contributors</Title>
                <Divider />
                {detail.contributors.length === 0 ? (
                    <Text size="sm" c="dimmed">
                        No device matched
                    </Text>
                ) : (
                    detail.contributors.map((c, i) => (
                        <Group key={i} gap="xs" align="flex-start">
                            <Text size="sm" ff="monospace">
                                {c.device_name ?? (c.device_id != null ? `Device #${c.device_id}` : "—")}
                            </Text>
                            {(c.user_name ?? c.user_id != null) && (
                                <Text size="sm" c="dimmed">
                                    · {c.user_name ?? `User #${c.user_id}`}
                                </Text>
                            )}
                        </Group>
                    ))
                )}
                {detail.contributor_count > 1 && (
                    <Text size="xs" c="dimmed">
                        This client IP resolved to {detail.contributor_count} contributors.
                    </Text>
                )}
            </Stack>

            {detail.country_code && (
                <Stack gap="xs">
                    <Title order={3} size="h5">Location</Title>
                    <Divider />
                    <LabelValue
                        label="Country"
                        value={`${countryFlagEmoji(detail.country_code)} ${detail.country_name ?? ""} (${detail.country_code})`.trim()}
                    />
                    {detail.continent_code && (
                        <LabelValue label="Continent" value={detail.continent_code} />
                    )}
                    {(detail.asn != null || detail.asn_org) && (
                        <LabelValue
                            label="ASN"
                            value={[detail.asn, detail.asn_org].filter(Boolean).join(" — ")}
                        />
                    )}
                </Stack>
            )}

            {headersJson && (
                <Stack gap="xs">
                    <Group justify="space-between">
                        <Title order={3} size="h5">Headers</Title>
                        <Tooltip label="Copy headers">
                            <ActionIcon
                                variant="subtle"
                                size="sm"
                                aria-label="Copy headers"
                                onClick={() => copy(headersJson, { successMessage: "Headers copied to clipboard", errorMessage: "Failed to copy headers" })}
                            >
                                <IconCopy size={14} />
                            </ActionIcon>
                        </Tooltip>
                    </Group>
                    <Divider />
                    {!detail.outcome && (
                        <Text size="xs" c="dimmed">
                            Denied requests retain only a subset of headers
                            (X-Forwarded-Host, -Uri, -Method, -Proto, -For, X-Real-Ip
                            and User-Agent), so rejected traffic cannot flood the log.
                        </Text>
                    )}
                    <Code block>{headersJson}</Code>
                </Stack>
            )}
        </Stack>
    );
}

function LabelValue({
    label,
    value,
    mono = false,
}: {
    label: string;
    value: string;
    mono?: boolean;
}) {
    return (
        <Group gap="xs" align="flex-start">
            <Text size="sm" fw={500} w={100} style={{ flexShrink: 0 }}>
                {label}
            </Text>
            <Text size="sm" ff={mono ? "monospace" : undefined} style={{ wordBreak: "break-all" }}>
                {value}
            </Text>
        </Group>
    );
}
