import type { FleetDevice } from "@/lib/api";
import { DeviceState } from "@/lib/api";
import { DeviceApiKeyRuleHintBanner } from "@/features/devices/DeviceApiKeyRuleHintBanner";
import { DeviceDisabledBanner } from "@/features/devices/DeviceDisabledBanner";

interface Props {
  device: FleetDevice;
  ownerId: number;
  onGoToRules: () => void;
  onGoToSettings: () => void;
}

/** Attention banners above the device workspace tabs. */
export function DeviceWorkspaceBanners({
  device,
  ownerId,
  onGoToRules,
  onGoToSettings,
}: Props) {
  const showApiKeyHint =
    Boolean(device.api_key_prefix) && !device.rules.some((r) => r.enabled);

  return (
    <>
      {device.state === DeviceState.DISABLED && (
        <DeviceDisabledBanner deviceId={device.id} ownerId={ownerId} />
      )}
      {showApiKeyHint && (
        <DeviceApiKeyRuleHintBanner
          deviceId={device.id}
          onGoToRules={onGoToRules}
          onGoToSettings={onGoToSettings}
        />
      )}
    </>
  );
}
