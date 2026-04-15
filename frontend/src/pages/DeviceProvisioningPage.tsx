import { useState } from "react";
import { Stack, Text, Title } from "@mantine/core";
import type { PendingRegistration } from "@/lib/api";
import { InviteCreationForm } from "@/features/provisioning/InviteCreationForm";
import { InviteDetailPanel } from "@/features/provisioning/InviteDetailPanel";
import { InviteList } from "@/features/provisioning/InviteList";

export function DeviceProvisioningPage() {
  const [createdInvite, setCreatedInvite] =
    useState<PendingRegistration | null>(null);

  return (
    <Stack maw={1024} gap="xl">
      <div>
        <Title order={1}>Device Provisioning</Title>
        <Text c="dimmed" size="sm">
          Generate setup codes for new devices.
        </Text>
      </div>
      {createdInvite ? (
        <InviteDetailPanel
          registration={createdInvite}
          onCreateAnother={() => setCreatedInvite(null)}
        />
      ) : (
        <InviteCreationForm onSuccess={setCreatedInvite} />
      )}
      <InviteList />
    </Stack>
  );
}
