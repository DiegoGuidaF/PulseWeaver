import { ActionIcon, Popover, SimpleGrid } from "@mantine/core";
import { HOST_ICON_OPTIONS } from "@/features/host-access/hostIconConfig";

interface Props {
  opened: boolean;
  onClose: () => void;
  target: React.ReactNode;
  selectedIcon: string;
  onSelect: (name: string) => void;
}

export function HostIconPickerPopover({ opened, onClose, target, selectedIcon, onSelect }: Props) {
  return (
    <Popover opened={opened} onClose={onClose} position="bottom-start" withinPortal shadow="md">
      <Popover.Target>{target}</Popover.Target>
      <Popover.Dropdown>
        <SimpleGrid cols={5} spacing={4}>
          {HOST_ICON_OPTIONS.map(({ name, icon: Icon }) => (
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
