import { useState } from "react";
import { Button, Group, Modal, Stack, Text, TextInput } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useUpdateKnownHost } from "@/features/host-access/hooks/useUpdateKnownHost";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  opened: boolean;
  onClose: () => void;
  hostId: number;
  hostFqdn: string;
  currentIcon: string | null;
}

export function EditHostIconModal({ opened, onClose, hostId, hostFqdn, currentIcon }: Props) {
  const [icon, setIcon] = useState(currentIcon ?? "");
  const updateKnownHost = useUpdateKnownHost();

  function handleSave() {
    updateKnownHost.mutate(
      { path: { host_id: hostId }, body: { icon: icon.trim() || null } },
      {
        onSuccess: () => {
          notifications.show({ color: "green", message: "Icon updated" });
          onClose();
        },
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to update icon", message: toErrorMessage(err) }),
      },
    );
  }

  return (
    <Modal opened={opened} onClose={onClose} title={`Edit icon — ${hostFqdn}`}>
      <Stack gap="md">
        <TextInput
          label="Icon"
          description="Tabler icon name (e.g. server, lock, cloud). Leave empty to clear."
          placeholder="e.g. server"
          value={icon}
          onChange={(e) => setIcon(e.currentTarget.value)}
          autoFocus
        />
        {icon.trim() && (
          <Text size="xs" c="dimmed" ff="monospace">
            ti ti-{icon.trim()}
          </Text>
        )}
        <Group justify="flex-end" gap="xs">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={updateKnownHost.isPending}
            loading={updateKnownHost.isPending}
          >
            Save
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
