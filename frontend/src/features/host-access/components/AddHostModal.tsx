import { useState } from "react";
import { Button, Group, Modal, Stack, TextInput } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useCreateKnownHosts } from "@/features/host-access/hooks/useCreateKnownHosts";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  opened: boolean;
  onClose: () => void;
}

export function AddHostModal({ opened, onClose }: Props) {
  const [fqdn, setFqdn] = useState("");
  const createKnownHosts = useCreateKnownHosts();

  function handleSubmit() {
    const value = fqdn.trim().toLowerCase();
    if (!value) return;
    createKnownHosts.mutate(
      { body: { fqdns: [value] } },
      {
        onSuccess: () => {
          notifications.show({ color: "green", message: `${value} added to known hosts` });
          setFqdn("");
          onClose();
        },
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to add host", message: toErrorMessage(err) }),
      },
    );
  }

  function handleClose() {
    setFqdn("");
    onClose();
  }

  return (
    <Modal opened={opened} onClose={handleClose} title="New known host">
      <Stack gap="md">
        <TextInput
          label="FQDN"
          description="Exact match — no wildcards."
          placeholder="e.g. jellyfin.myhome.org"
          value={fqdn}
          onChange={(e) => setFqdn(e.currentTarget.value)}
          ff="monospace"
          autoFocus
          onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
        />
        <Group justify="flex-end" gap="xs">
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!fqdn.trim() || createKnownHosts.isPending}
            loading={createKnownHosts.isPending}
          >
            Add host
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
