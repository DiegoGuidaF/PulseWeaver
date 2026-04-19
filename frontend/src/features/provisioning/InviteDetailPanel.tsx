import { ActionIcon, Button, Card, Group, Text, TextInput } from "@mantine/core";
import { IconCopy } from "@tabler/icons-react";
import type { PendingRegistration } from "@/lib/api";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { useClipboard } from "@/hooks/useClipboard";

interface InviteDetailPanelProps {
  registration: PendingRegistration;
  onCreateAnother?: () => void;
}

export function InviteDetailPanel({
  registration,
  onCreateAnother,
}: InviteDetailPanelProps) {
  const formatDateTime = useDateFormatter();
  const { copy } = useClipboard();

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
          onClick={() => copy(registration.registration_code ?? "", { successMessage: "Code copied" })}
          disabled={!registration.registration_code}
          aria-label="Copy registration code"
          mb={1}
        >
          <IconCopy size={16} />
        </ActionIcon>
      </Group>
      <Text size="sm" c="dimmed" mt="sm">
        Expires{" "}
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
