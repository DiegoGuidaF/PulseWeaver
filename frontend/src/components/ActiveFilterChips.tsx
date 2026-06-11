import { Button, Group, Pill } from "@mantine/core";
import { IconFilterOff } from "@tabler/icons-react";

export interface FilterChip {
    label: string;
    value: string;
    onRemove: () => void;
}

interface ActiveFilterChipsProps {
    chips: FilterChip[];
    /** When set, renders a "Clear filters" button at the end of the chip row. */
    onClearAll?: () => void;
}

export function ActiveFilterChips({ chips, onClearAll }: ActiveFilterChipsProps) {
    if (chips.length === 0) return null;

    return (
        <Pill.Group>
            <Group gap="xs">
                {chips.map((chip) => (
                    <Pill
                        key={`${chip.label}: ${chip.value}`}
                        withRemoveButton
                        onRemove={chip.onRemove}
                        size="sm"
                    >
                        {/* Plain span: Mantine Text sets `text-wrap: wrap`, which lets the
                            value wrap onto a line the pill clips away. */}
                        <span style={{ fontWeight: 600, color: "var(--mantine-color-dimmed)" }}>
                            {chip.label}:
                        </span>{" "}
                        {chip.value}
                    </Pill>
                ))}
                {onClearAll && (
                    <Button
                        variant="subtle"
                        size="compact-xs"
                        leftSection={<IconFilterOff size={14} />}
                        onClick={onClearAll}
                    >
                        Clear filters
                    </Button>
                )}
            </Group>
        </Pill.Group>
    );
}
