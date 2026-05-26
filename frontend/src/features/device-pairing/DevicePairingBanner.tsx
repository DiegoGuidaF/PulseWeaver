import dayjs from "dayjs";
import { Alert, Button, Group, Text } from "@mantine/core";
import { IconDeviceMobile } from "@tabler/icons-react";

function formatTtl(expiresAt: string): string {
  const diffMin = dayjs(expiresAt).diff(dayjs(), "minute");
  if (diffMin <= 0) return "expired";
  if (diffMin < 60) return `${diffMin}m remaining`;
  const h = Math.floor(diffMin / 60);
  const m = diffMin % 60;
  return m > 0 ? `${h}h ${m}m remaining` : `${h}h remaining`;
}

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
