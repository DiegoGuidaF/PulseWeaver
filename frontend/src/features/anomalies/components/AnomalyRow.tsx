import { useState } from "react";
import { ActionIcon, Button, Card, Collapse, Group, Stack, Text, Tooltip } from "@mantine/core";
import { IconCheck, IconChevronDown, IconChevronRight } from "@tabler/icons-react";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { AnomalyStatus, type Anomaly } from "@/lib/api";
import { ANOMALY_KIND_META } from "../constants";
import { SeverityIndicator } from "./SeverityIndicator";
import { AnomalyAttributionChips } from "./AnomalyAttributionChips";
import { EvidenceList } from "./EvidenceList";

dayjs.extend(relativeTime);

interface AnomalyRowProps {
    anomaly: Anomaly;
    /** Page rows expand to the raw evidence; the dashboard section keeps rows compact. */
    expandable?: boolean;
    /** "window" shows the absolute first→last-seen range (page); "relative" shows a `fromNow()` string (dashboard). */
    dateDisplay?: "relative" | "window";
    onAcknowledge?: (anomaly: Anomaly) => void;
    isAcknowledging?: boolean;
}

export function AnomalyRow({
    anomaly,
    expandable = false,
    dateDisplay = "relative",
    onAcknowledge,
    isAcknowledging = false,
}: AnomalyRowProps) {
    const [expanded, setExpanded] = useState(false);
    const formatDateTime = useDateFormatter();
    const kindMeta = ANOMALY_KIND_META[anomaly.kind];
    const Icon = kindMeta.icon;
    const isOpen = anomaly.status === AnomalyStatus.OPEN;

    const seenLabel =
        dateDisplay === "window"
            ? anomaly.first_seen_at === anomaly.last_seen_at
                ? formatDateTime(anomaly.last_seen_at)
                : `${formatDateTime(anomaly.first_seen_at)} → ${formatDateTime(anomaly.last_seen_at)}`
            : dayjs(anomaly.last_seen_at).fromNow();

    return (
        <Card withBorder p="sm" radius="md">
            <Stack gap={6}>
                <Group justify="space-between" align="flex-start" wrap="wrap" gap="xs">
                    <Group gap="xs" wrap="wrap" style={{ flex: 1, minWidth: 0 }}>
                        <SeverityIndicator severity={anomaly.severity} />
                        <Group gap={4} wrap="nowrap">
                            <Icon size={14} stroke={1.5} />
                            <Text size="xs" fw={500}>
                                {kindMeta.label}
                            </Text>
                        </Group>
                    </Group>
                    <Group gap="xs" wrap="nowrap">
                        <Text size="xs" c="dimmed" truncate="end" style={{ maxWidth: 260 }}>
                            {seenLabel}
                        </Text>
                        {isOpen && onAcknowledge && (
                            <Button
                                size="compact-xs"
                                variant="light"
                                leftSection={<IconCheck size={12} />}
                                loading={isAcknowledging}
                                onClick={() => onAcknowledge(anomaly)}
                            >
                                Acknowledge
                            </Button>
                        )}
                        {expandable && (
                            <Tooltip label={expanded ? "Hide evidence" : "Show evidence"} withArrow>
                                <ActionIcon
                                    variant="subtle"
                                    color="gray"
                                    size="sm"
                                    onClick={() => setExpanded((v) => !v)}
                                    aria-label={expanded ? "Hide evidence" : "Show evidence"}
                                >
                                    {expanded ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
                                </ActionIcon>
                            </Tooltip>
                        )}
                    </Group>
                </Group>

                <Text size="sm">{anomaly.summary}</Text>

                <AnomalyAttributionChips anomaly={anomaly} />

                {expandable && (
                    <Collapse expanded={expanded}>
                        <EvidenceList evidence={anomaly.evidence} />
                    </Collapse>
                )}
            </Stack>
        </Card>
    );
}
