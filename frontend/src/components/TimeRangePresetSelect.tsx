import { Group, NativeSelect, Tooltip } from "@mantine/core";
import { IconClock } from "@tabler/icons-react";
import { DATEPICKER_TIME_PRESETS } from "@/lib/timePresets";

const OPTIONS = DATEPICKER_TIME_PRESETS.map(({ key, label }) => ({ label, value: key }));

interface TimeRangePresetSelectProps {
    value: string | null;
    onChange: (key: string | null) => void;
}

export function TimeRangePresetSelect({ value, onChange }: TimeRangePresetSelectProps) {
    // Without an explicit "Custom" entry a native select silently displays the
    // first option when the value matches none, misreporting the active window.
    const isPreset = value !== null && OPTIONS.some((o) => o.value === value);
    return (
        <Group gap="xs">
            <Tooltip label="Time range" withArrow>
                <IconClock size={16} style={{ color: "var(--mantine-color-dimmed)", flexShrink: 0 }} />
            </Tooltip>
            <NativeSelect
                value={isPreset ? value : ""}
                onChange={(e) => onChange(e.target.value || null)}
                aria-label="Time range"
                data={isPreset ? OPTIONS : [{ label: "Custom", value: "", disabled: true }, ...OPTIONS]}
            />
        </Group>
    );
}
