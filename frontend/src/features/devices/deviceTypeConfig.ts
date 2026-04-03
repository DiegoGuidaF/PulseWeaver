import {
  IconBrandAndroid,
  IconBrandApple,
  IconCloud,
  IconCpu,
  IconDatabase,
  IconDeviceDesktop,
  IconDeviceGamepad,
  IconDeviceMobile,
  IconDeviceTablet,
  IconDeviceTv,
  IconDeviceWatch,
  IconDevices,
  IconDeviceLaptop,
  IconRouter,
  IconServer,
} from "@tabler/icons-react";
import type { MantineColor } from "@mantine/core";
import type { Icon as TablerIcon } from "@tabler/icons-react";

export type DeviceType = "static" | "mobile";

export const DEVICE_TYPE_CONFIG: Record<
  DeviceType,
  { icon: TablerIcon; color: MantineColor }
> = {
  static: { icon: IconDevices, color: "dimmed" },
  mobile: { icon: IconDeviceMobile, color: "blue" },
};

export const ICON_PICKER_OPTIONS: { name: string; icon: TablerIcon }[] = [
  { name: "IconDeviceMobile", icon: IconDeviceMobile },
  { name: "IconDeviceLaptop", icon: IconDeviceLaptop },
  { name: "IconDeviceDesktop", icon: IconDeviceDesktop },
  { name: "IconServer", icon: IconServer },
  { name: "IconCpu", icon: IconCpu },
  { name: "IconDatabase", icon: IconDatabase },
  { name: "IconRouter", icon: IconRouter },
  { name: "IconDeviceTablet", icon: IconDeviceTablet },
  { name: "IconDeviceTv", icon: IconDeviceTv },
  { name: "IconBrandAndroid", icon: IconBrandAndroid },
  { name: "IconBrandApple", icon: IconBrandApple },
  { name: "IconDeviceWatch", icon: IconDeviceWatch },
  { name: "IconDeviceGamepad", icon: IconDeviceGamepad },
  { name: "IconCloud", icon: IconCloud },
  { name: "IconDevices", icon: IconDevices },
];

const ICON_MAP = new Map(
  ICON_PICKER_OPTIONS.map(({ name, icon }) => [name, icon]),
);

export function getDeviceIcon(device: {
  device_type: string;
  icon?: string | null;
}): { Icon: TablerIcon; color: MantineColor } {
  if (device.icon) {
    const resolved = ICON_MAP.get(device.icon);
    if (resolved) return { Icon: resolved, color: "dimmed" };
  }
  const typeConfig = DEVICE_TYPE_CONFIG[device.device_type as DeviceType];
  if (typeConfig) return { Icon: typeConfig.icon, color: typeConfig.color };
  return { Icon: IconDevices, color: "dimmed" };
}
