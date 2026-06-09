import { Alert, Button, Group, Text } from "@mantine/core";
import { IconLock } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { useEnableDevice } from "@/features/devices/hooks/useEnableDevice";

interface Props {
  deviceId: number;
}

export function DeviceDisabledBanner({ deviceId }: Props) {
  const enableDevice = useEnableDevice();

  return (
    <Alert
      color="gray"
      icon={<IconLock size={18} stroke={1.5} />}
      title="Device frozen"
    >
      <Group justify="space-between" align="center">
        <Text size="sm">Address updates are blocked until re-enabled.</Text>
        <Button
          size="xs"
          variant="light"
          color="gray"
          disabled={enableDevice.isPending}
          loading={enableDevice.isPending}
          onClick={() =>
            enableDevice.mutate(
              { path: { device_id: deviceId } },
              {
                onSuccess: () =>
                  notifications.show({
                    color: "green",
                    message: "Device re-enabled — address updates are allowed again.",
                  }),
                onError: (err) =>
                  notifications.show({ color: "red", message: toErrorMessage(err) }),
              },
            )
          }
        >
          Re-enable
        </Button>
      </Group>
    </Alert>
  );
}
