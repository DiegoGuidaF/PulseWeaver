import { Box, Group, Stack, Text, Tooltip, UnstyledButton } from "@mantine/core";
import { GROUP_COLOR_PALETTE, type GroupColor } from "@/features/host-access/drafts/hostGroupsDraft";

interface Props {
  value: GroupColor | null;
  onChange: (next: GroupColor | null) => void;
  label?: string;
  description?: string;
}

export function GroupColorPicker({
  value,
  onChange,
  label = "Colour",
  description = "Used to render the group throughout the UI.",
}: Props) {
  return (
    <Stack gap={6}>
      <Box>
        <Text size="sm" fw={500}>
          {label}
        </Text>
        {description && (
          <Text size="xs" c="dimmed">
            {description}
          </Text>
        )}
      </Box>
      <Group gap={6}>
        {GROUP_COLOR_PALETTE.map((color) => {
          const selected = value === color;
          return (
            <Tooltip key={color} label={color} withArrow openDelay={300}>
              <UnstyledButton
                aria-label={color}
                aria-pressed={selected}
                onClick={() => onChange(selected ? null : color)}
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: 6,
                  background: `var(--mantine-color-${color}-5)`,
                  border: selected
                    ? "2px solid var(--mantine-color-text)"
                    : "2px solid transparent",
                  outlineOffset: 2,
                  cursor: "pointer",
                }}
              />
            </Tooltip>
          );
        })}
      </Group>
    </Stack>
  );
}
