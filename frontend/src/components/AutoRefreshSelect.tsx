import { Group, NativeSelect, Tooltip } from "@mantine/core";
import { IconRefresh } from "@tabler/icons-react";

const REFRESH_OPTIONS = [
    { label: "Off", value: 0 },
    { label: "1s", value: 1_000 },
    { label: "5s", value: 5_000 },
    { label: "15s", value: 15_000 },
    { label: "30s", value: 30_000 },
    { label: "1 min", value: 60_000 },
    { label: "5 min", value: 300_000 },
] as const;

interface AutoRefreshSelectProps {
    value: number;
    onChange: (ms: number) => void;
}

export function AutoRefreshSelect({ value, onChange }: AutoRefreshSelectProps) {
    return (
        <Group gap="xs">
            <Tooltip label="Auto-refresh" withArrow>
                <IconRefresh size={16} style={{ color: "var(--mantine-color-dimmed)", flexShrink: 0 }} />
            </Tooltip>
            <NativeSelect
                value={value}
                onChange={(e) => onChange(Number(e.target.value))}
                aria-label="Auto-refresh interval"
                data={REFRESH_OPTIONS.map((opt) => ({
                    label: opt.label,
                    value: String(opt.value),
                }))}
            />
        </Group>
    );
}
