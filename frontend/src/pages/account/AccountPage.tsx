import { useState } from "react";
import { Card, Stack, Text, Title } from "@mantine/core";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { AccountSettings } from "@/features/account/AccountSettings";

export function AccountPage() {
  const { user } = useAuth();
  const [dirty, setDirty] = useState(false);

  // Prompt native browser dialog on tab close / refresh when there are unsaved changes
  useUnsavedChangesGuard(dirty);

  return (
    <Stack maw={1024} gap="xl">
      <div>
        <Title order={1}>Account</Title>
        <Text c="dimmed" mt={4}>Manage your profile and password.</Text>
      </div>

      {user?.must_change_password && (
        <Card withBorder style={{ borderColor: "var(--mantine-color-yellow-6)" }}>
          <Title order={2} mb="xs">Password change required</Title>
          <Text size="sm" c="dimmed">
            You must set a new password before using the rest of the application.
          </Text>
        </Card>
      )}

      <AccountSettings onDirtyChange={setDirty} />
    </Stack>
  );
}
