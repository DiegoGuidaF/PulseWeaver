import { ActionIcon, Badge, Code, Divider, Drawer, Group, Stack, Text, Title, Tooltip } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import type { AccessLogRow } from "@/lib/api";
import { DENY_REASON_LABELS } from "../constants";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { countryFlagEmoji } from "@/lib/countryFlag";
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

    async function handleCopyHeaders() {
        if (!row?.headers) return;
        if (!("clipboard" in navigator) || !navigator.clipboard?.writeText) {
            notifications.show({ message: "Copy to clipboard is not supported in this browser.", color: "red" });
            return;
        }
        try {
            await navigator.clipboard.writeText(JSON.stringify(row.headers, null, 2));
            notifications.show({ message: "Headers copied to clipboard", color: "green" });
        } catch {
            notifications.show({ message: "Failed to copy headers", color: "red" });
        }
    }

    return (
        <Drawer
            opened={opened}
            onClose={onClose}
            position="right"
            size="lg"
            title={<Text fw={600}>Request Detail</Text>}
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
                        <Title order={5}>Request</Title>
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
                        <Title order={5}>Device</Title>
                        <Divider />
                        {row.device_name ? (
                            <LabelValue label="Device" value={row.device_name} />
                        ) : (
                            <Text size="sm" c="dimmed">
                                No device matched
                            </Text>
                        )}
                    </Stack>

                    {row.country_code && (
                      <Stack gap="xs">
                        <Title order={5}>Location</Title>
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

                    {row.headers && Object.keys(row.headers).length > 0 && (
                        <Stack gap="xs">
                            <Group justify="space-between" align="center">
                                <Title order={5}>Headers</Title>
                                <Tooltip label="Copy headers">
                                    <ActionIcon
                                        variant="subtle"
                                        size="sm"
                                        aria-label="Copy headers"
                                        onClick={handleCopyHeaders}
                                    >
                                        <IconCopy size={14} />
                                    </ActionIcon>
                                </Tooltip>
                            </Group>
                            <Divider />
                            <Code block>
                                {JSON.stringify(row.headers, null, 2)}
                            </Code>
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
