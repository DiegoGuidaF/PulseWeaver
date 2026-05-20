import { useState } from "react";
import {
  Badge,
  Button,
  Card,
  Group,
  Modal,
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

function roleBadgeColor(role: UserRole): string {
  if (role === UserRole.SUPERADMIN) return "violet";
  if (role === UserRole.ADMIN) return "indigo";
  return "gray";
}

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

  const { data: users, isLoading: usersLoading } = useListUsers({
    enabled: currentUser != null,
  });

  const updateDevice = useUpdateDevice();
  const [draftOwnerId, setDraftOwnerId] = useState<string | null>(null);
  const [confirmTarget, setConfirmTarget] = useState<{
    ownerId: number;
    ownerName: string;
    ownerRole: UserRole;
  } | null>(null);

  const serverValue = ownerId != null ? String(ownerId) : "";
  const selectedOwner = draftOwnerId ?? serverValue;
  const ownerDirty = draftOwnerId !== null && draftOwnerId !== serverValue;

  const pendingOwnerName = confirmTarget?.ownerName;
  const prevOwnerRole = users?.find((u) => u.id === ownerId)?.role;

  function handleOwnerSave() {
    if (!selectedOwner) return;
    const newUser = users?.find((u) => String(u.id) === selectedOwner);
    const newOwnerName = newUser?.display_name ?? selectedOwner;
    const newOwnerRole = newUser?.role ?? UserRole.USER;
    setConfirmTarget({ ownerId: Number(selectedOwner), ownerName: newOwnerName, ownerRole: newOwnerRole });
  }

  function handleConfirmOwnerChange() {
    if (!confirmTarget) return;
    updateDevice.mutate(
      {
        path: { device_id: deviceId },
        body: { owner_id: confirmTarget.ownerId },
      },
      {
        onSuccess: () => {
          setConfirmTarget(null);
          setDraftOwnerId(null);
          notifications.show({
            color: "green",
            message: "Device ownership updated",
          });
        },
        onError: (err) => {
          const status =
            err && typeof err === "object" && "status" in err
              ? (err as { status: unknown }).status
              : undefined;
          setConfirmTarget(null);
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
    <>
      <Modal
        opened={confirmTarget !== null}
        onClose={() => setConfirmTarget(null)}
        title="Reassign device ownership?"
        closeOnClickOutside={false}
      >
        <Stack gap="xs" my="xs">
          <Group gap="xs">
            <Text size="sm" c="dimmed" w={36}>From</Text>
            <Text size="sm" fw={500}>{ownerName ?? "—"}</Text>
            {prevOwnerRole && (
              <Badge variant="light" color={prevOwnerRole ? roleBadgeColor(prevOwnerRole) : "gray"} size="sm">
                {prevOwnerRole}
              </Badge>
            )}
          </Group>
          <Group gap="xs">
            <Text size="sm" c="dimmed" w={36}>To</Text>
            <Text size="sm" fw={500}>{pendingOwnerName}</Text>
            {confirmTarget?.ownerRole && (
              <Badge variant="light" color={roleBadgeColor(confirmTarget.ownerRole)} size="sm">
                {confirmTarget.ownerRole}
              </Badge>
            )}
          </Group>
        </Stack>
        <Group justify="flex-end" mt="md" gap="sm">
          <Button variant="outline" onClick={() => setConfirmTarget(null)}>
            Cancel
          </Button>
          <Button
            onClick={handleConfirmOwnerChange}
            disabled={updateDevice.isPending}
            loading={updateDevice.isPending}
          >
            Reassign
          </Button>
        </Group>
      </Modal>

      <Card withBorder>
      <Stack gap="md">
        {usersLoading ? (
          <Skeleton height={36} width={240} />
        ) : (
          <>
            <Select
              label="Owner"
              data={selectData}
              value={selectedOwner}
              onChange={(val) => setDraftOwnerId(val ?? "")}
              searchable
              w={300}
            />
            <Group gap="sm">
              {ownerDirty && (
                <Button
                  variant="subtle"
                  size="sm"
                  onClick={() => setDraftOwnerId(null)}
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
    </>
  );
}
