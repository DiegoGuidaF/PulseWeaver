import { ActionIcon, Popover, SimpleGrid } from "@mantine/core";
import { ICON_PICKER_OPTIONS } from "@/features/devices/deviceTypeConfig";

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
                onSelect(name);
                onClose();
              }}
            >
              <Icon size={18} />
            </ActionIcon>
          ))}
        </SimpleGrid>
      </Popover.Dropdown>
    </Popover>
  );
}
