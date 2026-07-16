import { Alert, Button, Group, Text } from "@mantine/core";
import { IconDeviceMobile } from "@tabler/icons-react";
import { formatTtl } from "./utils/formatTtl";

interface Props {
  expiresAt: string;
  onViewPairing: () => void;
}

export function DevicePairingBanner({ expiresAt, onViewPairing }: Props) {
  return (
    <Alert
      color="indigo"
      icon={<IconDeviceMobile size={18} stroke={1.5} />}
      title="Pairing code outstanding"
    >
      <Group justify="space-between" align="center">
        <Text size="sm">{formatTtl(expiresAt)} until expiry</Text>
        <Button size="xs" variant="light" color="indigo" onClick={onViewPairing}>
          View pairing →
        </Button>
      </Group>
    </Alert>
  );
}
