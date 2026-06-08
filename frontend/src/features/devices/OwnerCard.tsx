import { useNavigate } from "react-router-dom";
import { Avatar, Badge, Button, Card, Divider, Group, Stack, Text, UnstyledButton } from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import { buildRoute } from "@/lib/routes";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";
import { DeviceRow } from "@/features/devices/DeviceRow";
import type { DeviceListEntry, DeviceListOwner } from "@/lib/api";

function getInitials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

interface Props {
  owner: DeviceListOwner;
  devices: DeviceListEntry[];
}

export function OwnerCard({ owner, devices }: Props) {
  const navigate = useNavigate();
  const goCreate = () => navigate(buildRoute.userDevicesNew(owner.id));

  return (
      <Card withBorder radius="md" p="md">
        <Group justify="space-between" align="flex-start" mb="sm">
          <Group gap="sm" align="flex-start">
            <Avatar radius="xl" size="md" color="indigo">
              {getInitials(owner.display_name)}
            </Avatar>
            <Stack gap={2}>
              <Group gap="xs" align="center">
                <Text fw={600} size="sm">{owner.display_name}</Text>
                {owner.role === "admin" && (
                  <Badge size="xs" color="indigo" variant="light">admin</Badge>
                )}
              </Group>
              <Group gap={4} align="center" wrap="nowrap">
                {owner.bypass_host_check ? (
                  <Badge size="xs" color="orange" variant="filled">bypass</Badge>
                ) : owner.host_groups.length > 0 ? (
                  <GroupBadgeList groups={owner.host_groups} size="xs" />
                ) : null}
                <Text size="xs" c="dimmed">
                  {owner.device_count} device{owner.device_count !== 1 ? "s" : ""}
                  {owner.live_address_count > 0
                    ? ` · ${owner.live_address_count} live`
                    : ""}
                </Text>
              </Group>
            </Stack>
          </Group>

          <Button
            variant="default"
            size="xs"
            leftSection={<IconPlus size={13} />}
            onClick={goCreate}
          >
            Add device
          </Button>
        </Group>

        {devices.length > 0 ? (
          <>
            <Divider mb="xs" />
            <Stack gap={4}>
              {devices.map((entry) => (
                <DeviceRow key={entry.id} entry={entry} ownerId={owner.id} />
              ))}
            </Stack>
          </>
        ) : (
          <>
            <Divider mb="xs" />
            <UnstyledButton
              onClick={goCreate}
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                width: "100%",
                padding: "10px 0",
                borderRadius: "var(--mantine-radius-sm)",
                border: "1.5px dashed var(--mantine-color-default-border)",
                color: "var(--mantine-color-dimmed)",
                gap: 6,
              }}
            >
              <IconPlus size={13} />
              <Text size="xs" c="dimmed">add first device</Text>
            </UnstyledButton>
          </>
        )}
      </Card>
  );
}
