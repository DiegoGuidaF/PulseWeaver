import React from "react";
import { useState } from "react";
import {
  ActionIcon,
  Box,
  Group,
  Input,
  SimpleGrid,
  Stack,
  Text,
  ThemeIcon,
  Tooltip,
} from "@mantine/core";
import {
  HOST_ICON_OPTIONS,
  resolveHostIcon,
  validateIconInput,
} from "@/features/host-access/hostIconConfig";

interface Props {
  value: string | null;
  onChange: (next: string | null) => void;
  color?: string;
  label?: string;
  description?: string;
}

export function IconPicker({
  value,
  onChange,
  color = "gray",
  label = "Icon",
  description = "Pick a suggestion, or paste any emoji.",
}: Props) {
  const [draft, setDraft] = useState(value ?? "");

  const validation = validateIconInput(draft);
  const errorMsg = validation.ok ? null : validation.reason;

  function commitDraft(next: string) {
    const trimmed = next.trim();
    setDraft(next);
    const v = validateIconInput(next);
    if (!v.ok) return;
    onChange(trimmed === "" ? null : trimmed);
  }

  function selectSuggestion(name: string) {
    setDraft(name);
    onChange(name);
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

      <SimpleGrid cols={10} spacing={4}>
        {HOST_ICON_OPTIONS.map(({ name, icon: Icon }) => {
          const selected = value === name;
          return (
            <Tooltip key={name} label={name} withArrow openDelay={400}>
              <ActionIcon
                variant={selected ? "filled" : "subtle"}
                color={selected ? color : "gray"}
                size="lg"
                aria-label={name}
                aria-pressed={selected}
                onClick={() => selectSuggestion(name)}
              >
                <Icon size={18} stroke={1.5} />
              </ActionIcon>
            </Tooltip>
          );
        })}
      </SimpleGrid>

      <Group gap={8} align="flex-start" wrap="nowrap">
        <PreviewTile value={value} color={color} />
        <Input.Wrapper error={errorMsg} style={{ flex: 1 }}>
          <Input
            value={draft}
            onChange={(e) => commitDraft(e.currentTarget.value)}
            placeholder="Paste an emoji or pick above"
            aria-label="Free-form icon input"
            error={errorMsg !== null}
          />
        </Input.Wrapper>
      </Group>
    </Stack>
  );
}

interface PreviewTileProps {
  value: string | null;
  color: string;
}

function PreviewTile({ value, color }: PreviewTileProps) {
  const resolved = resolveHostIcon(value);
  if (resolved.kind === "tabler") {
    return (
      <ThemeIcon
        variant="light"
        color={color}
        size={36}
        radius="md"
        aria-label="Icon preview"
      >
        {React.createElement(resolved.icon, { size: 20, stroke: 1.5 })}
      </ThemeIcon>
    );
  }
  return (
    <ThemeIcon
      variant="light"
      color={color}
      size={36}
      radius="md"
      aria-label="Icon preview"
    >
      <Text size="lg" span>
        {resolved.value}
      </Text>
    </ThemeIcon>
  );
}
