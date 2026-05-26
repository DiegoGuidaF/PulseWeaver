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
import {
  EMOJI_RE,
  makeEmojiRenderer,
  makeTablerRenderer,
  validateIconWithMap,
} from "@/lib/iconUtils";
export type { IconRenderer, IconValidation } from "@/lib/iconUtils";

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

export function resolveDeviceIcon(icon?: string | null) {
  if (icon) {
    const tabler = ICON_MAP.get(icon);
    if (tabler) return makeTablerRenderer(tabler);
    if (EMOJI_RE.test(icon)) return makeEmojiRenderer(icon);
  }
  return makeTablerRenderer(IconDevices);
}

export function getDeviceIcon(device: {
  device_type: string;
  icon?: string | null;
}) {
  if (device.icon) {
    const tabler = ICON_MAP.get(device.icon);
    if (tabler) return makeTablerRenderer(tabler, { color: "var(--mantine-color-dimmed)" });
    if (EMOJI_RE.test(device.icon)) return makeEmojiRenderer(device.icon);
  }
  const typeConfig = DEVICE_TYPE_CONFIG[device.device_type as DeviceType];
  if (typeConfig) {
    const colorStyle =
      typeConfig.color === "dimmed"
        ? { color: "var(--mantine-color-dimmed)" }
        : { color: `var(--mantine-color-${typeConfig.color}-filled)` };
    return makeTablerRenderer(typeConfig.icon, colorStyle);
  }
  return makeTablerRenderer(IconDevices, { color: "var(--mantine-color-dimmed)" });
}

export function validateDeviceIconInput(raw: string) {
  return validateIconWithMap(ICON_MAP, raw);
}

// Maps keyword patterns to icon components. Names are derived from ICON_PICKER_OPTIONS
// by reference so there is no duplicated string to keep in sync.
const KEYWORD_ICON_MAP: Array<[RegExp, TablerIcon]> = [
  [/\b(phone|mobile|pixel|iphone|galaxy|android)\b/i, IconDeviceMobile],
  [/\b(laptop|mac|macbook|notebook)\b/i, IconDeviceLaptop],
  [/\b(desktop|workstation|imac)\b/i, IconDeviceDesktop],
  [/\b(server|vm|node|host)\b/i, IconServer],
  [/\b(router|ap|wifi|gateway)\b/i, IconRouter],
  [/\b(tablet|ipad)\b/i, IconDeviceTablet],
  [/\b(tv|television)\b/i, IconDeviceTv],
  [/\b(watch)\b/i, IconDeviceWatch],
  [/\b(gamepad|game|console)\b/i, IconDeviceGamepad],
];

export function suggestIcon(name: string): string {
  for (const [pattern, icon] of KEYWORD_ICON_MAP) {
    if (pattern.test(name)) {
      return ICON_PICKER_OPTIONS.find((o) => o.icon === icon)?.name ?? "";
    }
  }
  return "";
}
