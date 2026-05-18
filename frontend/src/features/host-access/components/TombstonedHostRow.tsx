import { ActionIcon, Badge, Group, Table, Text, Tooltip } from "@mantine/core";
import { IconArrowBackUp } from "@tabler/icons-react";
import type { Host } from "@/lib/api";

interface Props {
  host: Host;
  onRestore: () => void;
}

export function TombstonedHostRow({ host, onRestore }: Props) {
  return (
    <Table.Tr style={{ opacity: 0.55 }}>
      <Table.Td>
        <Group gap="xs" wrap="nowrap">
          <Text size="sm" fw={500} ff="monospace" td="line-through">
            {host.fqdn}
          </Text>
          <Badge size="xs" color="red" variant="light">
            Will delete
          </Badge>
        </Group>
      </Table.Td>
      <Table.Td colSpan={2} />
      <Table.Td>
        <Group gap={4} justify="flex-end">
          <Tooltip label="Undo delete" withArrow>
            <ActionIcon
              variant="subtle"
              size="sm"
              onClick={onRestore}
              aria-label={`Restore ${host.fqdn}`}
            >
              <IconArrowBackUp size={14} stroke={1.5} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Table.Td>
    </Table.Tr>
  );
}
