import { Table, Text } from "@mantine/core";

interface EvidenceListProps {
    evidence: Record<string, unknown>;
}

/**
 * Renders a raw evidence object as generic key–value pairs. The frontend
 * never interprets `evidence` per anomaly kind — that's the backend's
 * `summary` sentence's job — so values are formatted only by their JS type:
 * arrays joined with commas, objects JSON-stringified, everything else
 * stringified as-is.
 */
function formatEvidenceValue(value: unknown): string {
    if (value == null) return "—";
    if (Array.isArray(value)) return value.map((v) => String(v)).join(", ");
    if (typeof value === "object") return JSON.stringify(value);
    return String(value);
}

export function EvidenceList({ evidence }: EvidenceListProps) {
    const entries = Object.entries(evidence);
    if (entries.length === 0) {
        return (
            <Text size="xs" c="dimmed">
                No additional evidence recorded.
            </Text>
        );
    }

    return (
        <Table withRowBorders={false} verticalSpacing={4} horizontalSpacing="sm">
            <Table.Tbody>
                {entries.map(([key, value]) => (
                    <Table.Tr key={key}>
                        <Table.Td style={{ width: "40%" }}>
                            <Text size="xs" c="dimmed" ff="monospace">
                                {key}
                            </Text>
                        </Table.Td>
                        <Table.Td>
                            <Text size="xs" ff="monospace" style={{ wordBreak: "break-word" }}>
                                {formatEvidenceValue(value)}
                            </Text>
                        </Table.Td>
                    </Table.Tr>
                ))}
            </Table.Tbody>
        </Table>
    );
}
