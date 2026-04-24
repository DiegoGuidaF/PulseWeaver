import {
  IconApi,
  IconCloud,
  IconCode,
  IconDatabase,
  IconGlobe,
  IconKey,
  IconLink,
  IconLock,
  IconMail,
  IconRouter,
  IconServer,
  IconServer2,
  IconShieldLock,
  IconDeviceSdCard,
  IconWorldWww,
} from "@tabler/icons-react";
import type { Icon as TablerIcon } from "@tabler/icons-react";

export const HOST_ICON_OPTIONS: { name: string; icon: TablerIcon }[] = [
  { name: "IconServer", icon: IconServer },
  { name: "IconServer2", icon: IconServer2 },
  { name: "IconDatabase", icon: IconDatabase },
  { name: "IconCloud", icon: IconCloud },
  { name: "IconGlobe", icon: IconGlobe },
  { name: "IconWorldWww", icon: IconWorldWww },
  { name: "IconRouter", icon: IconRouter },
  { name: "IconDeviceSdCard", icon: IconDeviceSdCard },
  { name: "IconApi", icon: IconApi },
  { name: "IconCode", icon: IconCode },
  { name: "IconLink", icon: IconLink },
  { name: "IconLock", icon: IconLock },
  { name: "IconKey", icon: IconKey },
  { name: "IconShieldLock", icon: IconShieldLock },
  { name: "IconMail", icon: IconMail },
];

const HOST_ICON_MAP = new Map(HOST_ICON_OPTIONS.map(({ name, icon }) => [name, icon]));

export function getHostIcon(icon?: string | null): TablerIcon {
  if (icon) {
    const resolved = HOST_ICON_MAP.get(icon);
    if (resolved) return resolved;
  }
  return IconServer;
}
