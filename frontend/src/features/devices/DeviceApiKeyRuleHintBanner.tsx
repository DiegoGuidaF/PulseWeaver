import { Alert, Button, Group, Text } from "@mantine/core";
import { useLocalStorage } from "@mantine/hooks";
import { IconKey } from "@tabler/icons-react";

interface Props {
  deviceId: number;
  onGoToRules: () => void;
  onGoToSettings: () => void;
}

export function DeviceApiKeyRuleHintBanner({ deviceId, onGoToRules, onGoToSettings }: Props) {
  const [dismissed, setDismissed] = useLocalStorage({
    key: `pw.device.api-key-hint.dismissed.${deviceId}`,
    defaultValue: false,
    getInitialValueInEffect: false,
  });

  if (dismissed) return null;

  return (
    <Alert
      color="yellow"
      icon={<IconKey size={18} stroke={1.5} />}
      title="API key active with no address limits"
    >
      <Text size="sm" mb="sm">
        The API key lets the companion app automatically register addresses. Without an
        address lease or a max-addresses limit, old IPs accumulate indefinitely and are
        never cleaned up. Configure limits on the{" "}
        <Button
          variant="transparent"
          size="compact-sm"
          color="yellow"
          p={0}
          style={{ verticalAlign: "baseline" }}
          onClick={onGoToRules}
        >
          Rules tab
        </Button>
        . If the key is no longer in use, remove it on the{" "}
        <Button
          variant="transparent"
          size="compact-sm"
          color="yellow"
          p={0}
          style={{ verticalAlign: "baseline" }}
          onClick={onGoToSettings}
        >
          Settings tab
        </Button>
        .
      </Text>
      <Group gap="xs">
        <Button size="xs" variant="light" color="yellow" onClick={onGoToRules}>
          Configure limits →
        </Button>
        <Button size="xs" variant="light" color="yellow" onClick={onGoToSettings}>
          Remove API key →
        </Button>
        <Button
          size="xs"
          variant="subtle"
          color="gray"
          ml="auto"
          onClick={() => setDismissed(true)}
        >
          Dismiss
        </Button>
      </Group>
    </Alert>
  );
}
