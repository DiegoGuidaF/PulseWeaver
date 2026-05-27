import { useCallback, useState } from "react";
import {
  Button,
  Card,
  Group,
  Modal,
  Stack,
  Tabs,
  Text,
  Title,
} from "@mantine/core";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { AccountTab } from "@/features/settings/AccountTab";
import { PreferencesTab } from "@/features/settings/PreferencesTab";

export function SettingsPage() {
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState<string | null>("account");
  const [accountDirty, setAccountDirty] = useState(false);
  const [pendingTab, setPendingTab] = useState<string | null>(null);

  // Prompt native browser dialog on tab close / refresh when there are unsaved changes
  useUnsavedChangesGuard(accountDirty);

  const handleTabChange = useCallback(
    (value: string | null) => {
      if (accountDirty && activeTab === "account") {
        setPendingTab(value);
        return;
      }
      setActiveTab(value);
    },
    [accountDirty, activeTab],
  );

  function confirmTabSwitch() {
    setAccountDirty(false);
    setActiveTab(pendingTab);
    setPendingTab(null);
  }

  function cancelTabSwitch() {
    setPendingTab(null);
  }

  return (
    <>
      {/* Tab-switch unsaved changes modal */}
      <Modal
        opened={pendingTab !== null}
        onClose={cancelTabSwitch}
        title="Unsaved changes"
        closeOnClickOutside={false}
      >
        <Text size="sm">
          You have unsaved profile changes. Do you want to discard them?
        </Text>
        <Group justify="flex-end" mt="md" gap="sm">
          <Button variant="outline" onClick={cancelTabSwitch}>
            Keep editing
          </Button>
          <Button color="red" onClick={confirmTabSwitch}>
            Discard changes
          </Button>
        </Group>
      </Modal>

      <Stack maw={1024} gap="xl">
        <div>
          <Title order={1}>Settings</Title>
          <Text c="dimmed">Manage your account and preferences.</Text>
        </div>

        {user?.must_change_password && (
          <Card withBorder style={{ borderColor: "var(--mantine-color-yellow-6)" }}>
            <Title order={2} mb="xs">Password change required</Title>
            <Text size="sm" c="dimmed">
              You must set a new password before using the rest of the application.
            </Text>
          </Card>
        )}

        <Tabs value={activeTab} onChange={handleTabChange} keepMounted={false}>
          <Tabs.List>
            <Tabs.Tab value="account">Account</Tabs.Tab>
            <Tabs.Tab value="preferences">Preferences</Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="account" pt="md">
            <AccountTab onDirtyChange={setAccountDirty} />
          </Tabs.Panel>

          <Tabs.Panel value="preferences" pt="md">
            <PreferencesTab />
          </Tabs.Panel>
        </Tabs>
      </Stack>
    </>
  );
}
