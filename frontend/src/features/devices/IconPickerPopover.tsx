import React, { useState } from "react";
import { ActionIcon, Input, Popover, SimpleGrid } from "@mantine/core";
import {
  ICON_PICKER_OPTIONS,
  validateDeviceIconInput,
} from "@/features/devices/deviceTypeConfig";

export interface IconPickerPopoverProps {
  opened: boolean;
  onClose: () => void;
  target: React.ReactNode;
  selectedIcon: string;
  onSelect: (name: string) => void;
}

export function IconPickerPopover({
  opened,
  onClose,
  target,
  selectedIcon,
  onSelect,
}: IconPickerPopoverProps) {
  const isEmoji = selectedIcon !== "" && !ICON_PICKER_OPTIONS.some((o) => o.name === selectedIcon);
  const [draft, setDraft] = useState(isEmoji ? selectedIcon : "");

  const validation = validateDeviceIconInput(draft);
  const errorMsg = validation.ok ? null : validation.reason;

  function commitDraft(next: string) {
    setDraft(next);
    const trimmed = next.trim();
    const v = validateDeviceIconInput(trimmed);
    if (!v.ok || trimmed === "") return;
    const isTabler = ICON_PICKER_OPTIONS.some((o) => o.name === trimmed);
    if (!isTabler) onSelect(trimmed);
  }

  return (
    <Popover
      opened={opened}
      onClose={onClose}
      position="bottom-start"
      withinPortal
      shadow="md"
    >
      <Popover.Target>{target}</Popover.Target>
      <Popover.Dropdown>
        <SimpleGrid cols={5} spacing={4}>
          {ICON_PICKER_OPTIONS.map(({ name, icon: Icon }) => (
            <ActionIcon
              key={name}
              variant={selectedIcon === name ? "filled" : "subtle"}
              size="lg"
              aria-label={name}
              onClick={() => {
                setDraft("");
                onSelect(name);
                onClose();
              }}
            >
              {React.createElement(Icon, { size: 18 })}
            </ActionIcon>
          ))}
        </SimpleGrid>

        <Input.Wrapper error={errorMsg} mt={8}>
          <Input
            value={draft}
            onChange={(e) => commitDraft(e.currentTarget.value)}
            placeholder="or paste an emoji"
            size="xs"
            error={errorMsg !== null}
          />
        </Input.Wrapper>
      </Popover.Dropdown>
    </Popover>
  );
}
