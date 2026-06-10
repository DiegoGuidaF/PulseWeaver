import { Link, useNavigate } from "react-router-dom";
import { buildRoute } from "@/lib/routes";
import {
  Anchor,
  Badge,
  Button,
  Group,
  Stack,
  Table,
  Text,
  ThemeIcon,
} from "@mantine/core";
import { IconDevices } from "@tabler/icons-react";
import { EmptyState } from "@/components/EmptyState";
import type { DeviceListItem } from "@/lib/api";
import { resolveDeviceIcon } from "@/features/devices/deviceTypeConfig";

interface UserDevicesTabProps {
  userId: number;
  devices: DeviceListItem[];
}

export function UserDevicesTab({ userId, devices }: UserDevicesTabProps) {
  const navigate = useNavigate();

  if (devices.length === 0) {
    return (
      <EmptyState
        icon={IconDevices}
        title="No devices yet."
        description="Set one up on the device page — create it now and provision a credential (API key or pairing code) whenever the user's ready."
        action={
          <Button component={Link} to={buildRoute.userDevicesNew(userId)} variant="light">
            Set up a device
          </Button>
        }
      />
    );
  }

  return (
    <Stack gap="sm">
      <Table fz="sm" withRowBorders highlightOnHover>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Device name</Table.Th>
            <Table.Th>Live IPs</Table.Th>
            <Table.Th>API key</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {devices.map((device) => {
            const deviceHref = `${buildRoute.userDevices(userId)}?device=${device.id}`;
            return (
              <Table.Tr
                key={device.id}
                style={{ cursor: "pointer" }}
                onClick={() => navigate(deviceHref)}
              >
                <Table.Td fw={500}>
                  <Anchor
                    component={Link}
                    to={deviceHref}
                    c="inherit"
                    underline="hover"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Group gap="xs" wrap="nowrap">
                      <ThemeIcon variant="transparent" size="sm" c="dimmed">
                        {resolveDeviceIcon(device.icon)({ size: 16 })}
                      </ThemeIcon>
                      {device.name}
                    </Group>
                  </Anchor>
                </Table.Td>
                <Table.Td c="dimmed">{device.live_address_count}</Table.Td>
                <Table.Td>
                  {device.api_key_prefix ? (
                    <Badge size="xs" variant="light" color="orange" ff="monospace">
                      ● {device.api_key_prefix}…
                    </Badge>
                  ) : (
                    <Text size="sm" c="dimmed">
                      —
                    </Text>
                  )}
                </Table.Td>
              </Table.Tr>
            );
          })}
        </Table.Tbody>
      </Table>
      <Group justify="flex-end">
        <Anchor component={Link} to={buildRoute.userDevices(userId)} size="sm">
          Manage all devices →
        </Anchor>
      </Group>
    </Stack>
  );
}
