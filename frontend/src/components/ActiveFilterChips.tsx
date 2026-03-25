import { Group, Pill, Text } from "@mantine/core";

export interface FilterChip {
    label: string;
    value: string;
    onRemove: () => void;
}

interface ActiveFilterChipsProps {
    chips: FilterChip[];
}

export function ActiveFilterChips({ chips }: ActiveFilterChipsProps) {
    if (chips.length === 0) return null;

    return (
        <Pill.Group>
            <Group gap="xs">
                {chips.map((chip) => (
                    <Pill
                        key={chip.label}
                        withRemoveButton
                        onRemove={chip.onRemove}
                        size="sm"
                    >
                        <Text component="span" size="xs" fw={600} c="dimmed">
                            {chip.label}:
                        </Text>{" "}
                        {chip.value}
                    </Pill>
                ))}
            </Group>
        </Pill.Group>
    );
}
