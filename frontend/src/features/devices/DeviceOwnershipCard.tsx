import { useEffect, useState } from "react";
import {
  Button,
  Card,
  Group,
  Select,
  Skeleton,
  Stack,
  Text,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { UserRole } from "@/lib/api";
import { useUpdateDevice } from "@/features/devices/hooks/useUpdateDevice";

export interface DeviceOwnershipCardProps {
  deviceId: number;
  ownerId?: number;
  ownerName?: string;
}

export function DeviceOwnershipCard({
  deviceId,
  ownerId,
  ownerName,
}: DeviceOwnershipCardProps) {
  const { data: currentUser } = useCurrentUser();
  const isAdmin = currentUser?.role === UserRole.ADMIN;

  const { data: users, isLoading: usersLoading } = useListUsers({
    enabled: isAdmin,
  });

  const updateDevice = useUpdateDevice(deviceId);
  const [selectedOwner, setSelectedOwner] = useState(
    ownerId != null ? String(ownerId) : "",
  );

  const ownerDirty =
    selectedOwner !== (ownerId != null ? String(ownerId) : "");

  // Sync with latest server state, but never overwrite an in-progress edit.
  useEffect(() => {
    if (ownerDirty) return;
    setSelectedOwner(ownerId != null ? String(ownerId) : "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ownerId]);

  function handleOwnerSave() {
    if (!selectedOwner) return;
    updateDevice.mutate(
      {
        path: { device_id: deviceId },
        body: { owner_id: Number(selectedOwner) },
      },
      {
        onSuccess: () =>
          notifications.show({
            color: "green",
            message: "Device ownership updated",
          }),
        onError: (err) => {
          const status =
            err && typeof err === "object" && "status" in err
              ? (err as { status: unknown }).status
              : undefined;
          notifications.show({
            color: "red",
            message:
              status === 403
                ? "Admin permission required to reassign ownership"
                : toErrorMessage(err),
          });
        },
      },
    );
  }

  const selectData =
    users?.map((u) => ({ value: String(u.id), label: u.display_name })) ?? [];

  return (
    <Card withBorder>
      <Stack gap="md">
        <Text fw={500}>Ownership</Text>
        {!isAdmin ? (
          <Group gap="xs">
            <Text size="sm" c="dimmed">
              Owned by
            </Text>
            <Text size="sm">{ownerName ?? "—"}</Text>
          </Group>
        ) : usersLoading ? (
          <Skeleton height={36} width={240} />
        ) : (
          <>
            <Select
              label="Owner"
              data={selectData}
              value={selectedOwner}
              onChange={(val) => setSelectedOwner(val ?? "")}
              searchable
              w={300}
            />
            <Group gap="sm">
              {ownerDirty && (
                <Button
                  variant="subtle"
                  size="sm"
                  onClick={() =>
                    setSelectedOwner(ownerId != null ? String(ownerId) : "")
                  }
                >
                  Reset
                </Button>
              )}
              <Button
                size="sm"
                disabled={!ownerDirty || updateDevice.isPending}
                loading={updateDevice.isPending}
                onClick={handleOwnerSave}
              >
                Save
              </Button>
            </Group>
          </>
        )}
      </Stack>
    </Card>
  );
}
