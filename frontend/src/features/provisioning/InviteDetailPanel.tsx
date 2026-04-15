import { ActionIcon, Button, Card, Group, Text, TextInput } from "@mantine/core";
import { IconCopy } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import type { PendingRegistration } from "@/lib/api";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";

interface InviteDetailPanelProps {
  registration: PendingRegistration;
  onCreateAnother?: () => void;
}

export function InviteDetailPanel({
  registration,
  onCreateAnother,
}: InviteDetailPanelProps) {
  const formatDateTime = useDateFormatter();

  function copyCode() {
    navigator.clipboard.writeText(registration.registration_code ?? "");
    notifications.show({ color: "green", message: "Code copied" });
  }

  return (
    <Card withBorder>
      <Text fw={500} mb="sm">
        Invite ready for &ldquo;{registration.device_name}&rdquo;
      </Text>
      <Group gap="xs" align="flex-end">
        <TextInput
          label="Registration code"
          readOnly
          value={registration.registration_code ?? ""}
          style={{ flex: 1 }}
          ff="monospace"
        />
        <ActionIcon
          variant="default"
          size="lg"
          onClick={copyCode}
          disabled={!registration.registration_code}
          aria-label="Copy registration code"
          mb={1}
        >
          <IconCopy size={16} />
        </ActionIcon>
      </Group>
      <Text size="sm" c="dimmed" mt="sm">
        Key: {registration.device_api_key_prefix} · Expires{" "}
        {formatDateTime(registration.expires_at)}
      </Text>
      {onCreateAnother && (
        <Button variant="subtle" mt="md" onClick={onCreateAnother}>
          Create another
        </Button>
      )}
    </Card>
  );
}
