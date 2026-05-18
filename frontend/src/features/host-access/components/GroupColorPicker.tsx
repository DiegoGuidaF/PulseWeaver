import { useState } from "react";
import { Box, Group, Stack, Text, TextInput, Tooltip, UnstyledButton } from "@mantine/core";
import { GROUP_COLOR_PALETTE } from "@/features/host-access/utils/groupColor";

const HEX_RE = /^#[0-9a-fA-F]{6}$/;

interface Props {
  value: string;
  onChange: (next: string) => void;
  label?: string;
  description?: string;
}

export function GroupColorPicker({
  value,
  onChange,
  label = "Colour",
  description = "Used to render the group throughout the UI.",
}: Props) {
  const [inputDraft, setInputDraft] = useState(value);

  function handleSwatchClick(color: string) {
    setInputDraft(color);
    onChange(color);
  }

  function handleInputChange(raw: string) {
    setInputDraft(raw);
    if (HEX_RE.test(raw)) onChange(raw);
  }

  function handleInputBlur() {
    if (!HEX_RE.test(inputDraft)) setInputDraft(value);
  }

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
                onClick={() => handleSwatchClick(color)}
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: 6,
                  background: color,
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
      <Group gap={8} align="flex-start">
        <Box
          style={{
            width: 28,
            height: 28,
            borderRadius: 6,
            flexShrink: 0,
            marginTop: 1,
            background: HEX_RE.test(value) ? value : "var(--mantine-color-default-border)",
            border: "2px solid var(--mantine-color-default-border)",
          }}
        />
        <TextInput
          value={inputDraft}
          onChange={(e) => handleInputChange(e.currentTarget.value)}
          onBlur={handleInputBlur}
          placeholder="#4C6EF5"
          size="xs"
          style={{ flex: 1 }}
          aria-label="Custom hex colour"
          error={
            inputDraft.length > 0 && !HEX_RE.test(inputDraft)
              ? "Enter a hex colour like #4C6EF5"
              : null
          }
        />
      </Group>
    </Stack>
  );
}
