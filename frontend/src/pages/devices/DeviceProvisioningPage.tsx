import { useState } from "react";
import { Button, Group, Modal, Stack, Text, Title } from "@mantine/core";
import type { PendingRegistration } from "@/lib/api";
import { InviteCreationForm } from "@/features/provisioning/InviteCreationForm";
import { InviteDetailPanel } from "@/features/provisioning/InviteDetailPanel";
import { InviteList } from "@/features/provisioning/InviteList";

export function DeviceProvisioningPage() {
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createdInvite, setCreatedInvite] = useState<PendingRegistration | null>(null);

  function handleCloseModal() {
    setCreateModalOpen(false);
    setCreatedInvite(null);
  }

  return (
    <Stack maw={1024} gap="xl">
      <Group justify="space-between">
        <div>
          <Title order={1}>Device Provisioning</Title>
          <Text c="dimmed" size="sm">
            Generate setup codes for new devices.
          </Text>
        </div>
        <Button onClick={() => setCreateModalOpen(true)}>Create invite</Button>
      </Group>

      <InviteList />

      <Modal
        opened={createModalOpen}
        onClose={handleCloseModal}
        title={createdInvite ? "Invite created" : "Create invite"}
        size="md"
      >
        {createdInvite ? (
          <InviteDetailPanel
            registration={createdInvite}
            onCreateAnother={() => setCreatedInvite(null)}
          />
        ) : (
          <InviteCreationForm
            onSuccess={setCreatedInvite}
            onCancel={handleCloseModal}
          />
        )}
      </Modal>
    </Stack>
  );
}
