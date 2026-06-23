import { ActionIcon, Badge, Code, Divider, Drawer, Group, Stack, Text, Title, Tooltip } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { useMemo } from "react";
import type { AccessLogRow } from "@/lib/api";
import { DENY_REASON_LABELS } from "../constants";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { countryFlagEmoji } from "@/lib/countryFlag";
import { useClipboard } from "@/hooks/useClipboard";
import { IconCopy } from "@tabler/icons-react";

interface AccessLogDetailDrawerProps {
    row: AccessLogRow | null;
    opened: boolean;
    onClose: () => void;
}

export function AccessLogDetailDrawer({
    row,
    opened,
    onClose,
}: AccessLogDetailDrawerProps) {
    const formatDateTime = useDateFormatter();
    const { copy } = useClipboard();
    // A fixed-width "lg" drawer is wider than a phone viewport and leaves its
    // header clipped above the visible area; fill the screen below the tablet
    // breakpoint so the title and close button stay reachable.
    const isMobile = useMediaQuery("(max-width: 48em)", false, { getInitialValueInEffect: false });
    const headersJson = useMemo(
        () => (row?.headers ? JSON.stringify(row.headers, null, 2) : null),
        [row],
    );

    return (
        <Drawer
            opened={opened}
            onClose={onClose}
            position="right"
            size={isMobile ? "100%" : "lg"}
            title={<Title order={2} size="h4">Request Detail</Title>}
        >
            {row && (
                <Stack gap="md">
                    <Group gap="xs">
                        <Badge color={row.outcome ? "green" : "red"} size="lg">
                            {row.outcome ? "Allowed" : "Denied"}
                        </Badge>
                        {!row.outcome && row.deny_reason && (
                            <Badge color="gray" variant="light" size="lg">
                                {DENY_REASON_LABELS[row.deny_reason] ?? row.deny_reason}
                            </Badge>
                        )}
                    </Group>

                    <Stack gap="xs">
                        <Title order={3} size="h5">Request</Title>
                        <Divider />
                        <LabelValue label="Time" value={formatDateTime(row.created_at)} />
                        <LabelValue label="Client IP" value={row.client_ip} mono />
                        {row.xff_chain && (
                            <LabelValue label="XFF Chain" value={row.xff_chain} mono />
                        )}
                        {row.http_method && (
                            <LabelValue label="Method" value={row.http_method} />
                        )}
                        {row.target_host && (
                            <LabelValue label="Host" value={row.target_host} mono />
                        )}
                        {row.target_uri && (
                            <LabelValue label="URI" value={row.target_uri} mono />
                        )}
                    </Stack>

                    <Stack gap="xs">
                        <Title order={3} size="h5">Contributors</Title>
                        <Divider />
                        {row.contributors.length === 0 ? (
                            <Text size="sm" c="dimmed">
                                No device matched
                            </Text>
                        ) : (
                            row.contributors.map((c, i) => (
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
                        {row.contributor_count > 1 && (
                            <Text size="xs" c="dimmed">
                                This client IP resolved to {row.contributor_count} contributors.
                            </Text>
                        )}
                    </Stack>

                    {row.country_code && (
                      <Stack gap="xs">
                        <Title order={3} size="h5">Location</Title>
                        <Divider />
                        <LabelValue
                          label="Country"
                          value={`${countryFlagEmoji(row.country_code)} ${row.country_name ?? ""} (${row.country_code})`.trim()}
                        />
                        {row.continent_code && (
                          <LabelValue label="Continent" value={row.continent_code} />
                        )}
                        {(row.asn != null || row.asn_org) && (
                          <LabelValue
                            label="ASN"
                            value={[row.asn, row.asn_org].filter(Boolean).join(" — ")}
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
                            {!row.outcome && (
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
            )}
        </Drawer>
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
