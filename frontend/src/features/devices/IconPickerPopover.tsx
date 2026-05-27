import React, { useState } from "react";
import {
  Button,
  Divider,
  Group,
  Input,
  Popover,
  SimpleGrid,
  Stack,
  Text,
  UnstyledButton,
} from "@mantine/core";
import { EMOJI_RE } from "@/lib/iconUtils";
import {
  getSuggestedEmoji,
  validateDeviceIconInput,
} from "@/features/devices/deviceTypeConfig";

export interface IconPickerPopoverProps {
  opened: boolean;
  onClose: () => void;
  target: React.ReactNode;
  selectedIcon: string;
  onSelect: (value: string) => void;
  /** Device name used to pick the relevant emoji bucket. */
  deviceName?: string;
}

const SECTION_LABEL_STYLE: React.CSSProperties = {
  fontFamily: "var(--mantine-font-family-monospace)",
  fontSize: 10,
  letterSpacing: "0.08em",
  textTransform: "uppercase",
};

function isCustomValue(value: string, suggestions: string[]): boolean {
  if (!value) return false;
  if (suggestions.includes(value)) return false;
  return EMOJI_RE.test(value) || value.startsWith("https://");
}

// ── Inner component ──────────────────────────────────────────────────────────
// Rendered only while the popover is open, so useState initialises fresh on
// each open — no useEffect needed to reset the draft.

interface PickerContentProps {
  selectedIcon: string;
  suggestions: string[];
  deviceName: string;
  onSelect: (value: string) => void;
  onClose: () => void;
}

function PickerContent({
  selectedIcon,
  suggestions,
  deviceName,
  onSelect,
  onClose,
}: PickerContentProps) {
  const [draft, setDraft] = useState(() =>
    isCustomValue(selectedIcon, suggestions) ? selectedIcon : "",
  );

  const validation = validateDeviceIconInput(draft);
  const errorMsg = validation.ok ? null : validation.reason;

  function handleUseDraft() {
    const trimmed = draft.trim();
    if (!trimmed || errorMsg) return;
    onSelect(trimmed);
    onClose();
  }

  return (
    <Stack gap={10}>
      {/* ── Suggestions ── */}
      <Stack gap={6}>
        <Text c="dimmed" style={SECTION_LABEL_STYLE}>
          {deviceName
            ? `Suggestions based on "${deviceName}"`
            : "Suggestions"}
        </Text>
        <SimpleGrid cols={6} spacing={6}>
          {suggestions.map((emoji) => {
            const selected = selectedIcon === emoji;
            return (
              <UnstyledButton
                key={emoji}
                onClick={() => {
                  onSelect(emoji);
                  onClose();
                }}
                aria-label={emoji}
                aria-pressed={selected}
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  width: 48,
                  height: 48,
                  borderRadius: "var(--mantine-radius-md)",
                  border: selected
                    ? "2px dashed var(--mantine-color-orange-4)"
                    : "1px solid var(--mantine-color-default-border)",
                  background: selected
                    ? "color-mix(in srgb, var(--mantine-color-orange-4) 12%, transparent)"
                    : undefined,
                  cursor: "pointer",
                  fontSize: 24,
                  lineHeight: 1,
                  transition: "border-color 120ms, background 120ms",
                }}
              >
                {emoji}
              </UnstyledButton>
            );
          })}
        </SimpleGrid>
      </Stack>

      <Divider variant="dashed" />

      {/* ── Custom ── */}
      <Stack gap={6}>
        <Text c="dimmed" style={SECTION_LABEL_STYLE}>
          Custom
        </Text>
        <Group gap={8} align="flex-start" wrap="nowrap">
          <Input.Wrapper error={errorMsg} style={{ flex: 1 }}>
            <Input
              value={draft}
              onChange={(e) => setDraft(e.currentTarget.value)}
              placeholder="paste emoji or URL…"
              size="sm"
              error={errorMsg !== null}
              radius="xl"
              onKeyDown={(e) => {
                if (e.key === "Enter") handleUseDraft();
              }}
            />
          </Input.Wrapper>
          <Button
            size="sm"
            variant="outline"
            radius="xl"
            disabled={!draft.trim() || !!errorMsg}
            onClick={handleUseDraft}
          >
            use
          </Button>
        </Group>
        <Text size="xs" c="dimmed">
          an emoji, a single-character glyph, or an https URL to a square
          image.
        </Text>
      </Stack>
    </Stack>
  );
}

// ── Public component ─────────────────────────────────────────────────────────

export function IconPickerPopover({
  opened,
  onClose,
  target,
  selectedIcon,
  onSelect,
  deviceName = "",
}: IconPickerPopoverProps) {
  const suggestions = getSuggestedEmoji(deviceName);

  return (
    <Popover
      opened={opened}
      onClose={onClose}
      position="bottom-start"
      withinPortal
      shadow="md"
      width={356}
    >
      <Popover.Target>{target}</Popover.Target>

      <Popover.Dropdown>
        {opened && (
          <PickerContent
            selectedIcon={selectedIcon}
            suggestions={suggestions}
            deviceName={deviceName}
            onSelect={onSelect}
            onClose={onClose}
          />
        )}
      </Popover.Dropdown>
    </Popover>
  );
}
