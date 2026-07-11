import { Box, Group, Text } from "@mantine/core";
import type { AnomalySeverity } from "@/lib/api";
import { ANOMALY_SEVERITY_META } from "../constants";

interface SeverityIndicatorProps {
    severity: AnomalySeverity;
}

/** Colored dot + label, per the UI style guide's severity registers (see `constants.ts`). */
export function SeverityIndicator({ severity }: SeverityIndicatorProps) {
    const meta = ANOMALY_SEVERITY_META[severity];
    return (
        <Group gap={6} wrap="nowrap" style={{ flexShrink: 0 }}>
            <Box
                style={{
                    width: 8,
                    height: 8,
                    borderRadius: "50%",
                    flexShrink: 0,
                    backgroundColor: `var(--mantine-color-${meta.color}-6)`,
                }}
            />
            <Text size="xs" fw={600} c={meta.color}>
                {meta.label}
            </Text>
        </Group>
    );
}
