import { Drawer, Stack, Text, Group, Badge, Code, Title, Divider } from "@mantine/core";
import type { RequestAuditLogRow } from "@/lib/api";
import { DENY_REASON_LABELS } from "../constants";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";

interface RequestAuditLogDetailDrawerProps {
    row: RequestAuditLogRow | null;
    opened: boolean;
    onClose: () => void;
}

export function RequestAuditLogDetailDrawer({
    row,
    opened,
    onClose,
}: RequestAuditLogDetailDrawerProps) {
    const formatDateTime = useDateFormatter();
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
                            <Badge color="orange" variant="light" size="lg">
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

                    {row.headers && Object.keys(row.headers).length > 0 && (
                        <Stack gap="xs">
                            <Title order={5}>Headers</Title>
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
