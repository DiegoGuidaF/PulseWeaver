import { Alert, Button, Stack, Text } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";

interface Props {
  title: string;
  message: string;
  discardLabel: string;
  onDiscard: () => void;
}

export function TabLockAlert({ title, message, discardLabel, onDiscard }: Props) {
  return (
    <Alert icon={<IconAlertCircle size={16} />} color="orange" title={title}>
      <Stack gap="sm">
        <Text size="sm">{message}</Text>
        <Button size="xs" variant="outline" color="orange" onClick={onDiscard} w="fit-content">
          {discardLabel}
        </Button>
      </Stack>
    </Alert>
  );
}
