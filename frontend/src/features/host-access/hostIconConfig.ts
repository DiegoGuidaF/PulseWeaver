import {
  IconApi,
  IconBooks,
  IconCloud,
  IconCode,
  IconDatabase,
  IconDeviceSdCard,
  IconDeviceTv,
  IconFile,
  IconGlobe,
  IconHome,
  IconKey,
  IconLink,
  IconLock,
  IconMail,
  IconMusic,
  IconPhoto,
  IconRouter,
  IconServer,
  IconServer2,
  IconShieldLock,
  IconTool,
  IconWorldWww,
} from "@tabler/icons-react";
import type { Icon as TablerIcon } from "@tabler/icons-react";
import {
  EMOJI_RE,
  makeEmojiRenderer,
  makeTablerRenderer,
  validateIconWithMap,
} from "@/lib/iconUtils";
export type { IconRenderer, IconValidation } from "@/lib/iconUtils";

export const HOST_ICON_OPTIONS: { name: string; icon: TablerIcon }[] = [
  { name: "IconServer", icon: IconServer },
  { name: "IconServer2", icon: IconServer2 },
  { name: "IconDatabase", icon: IconDatabase },
  { name: "IconCloud", icon: IconCloud },
  { name: "IconGlobe", icon: IconGlobe },
  { name: "IconWorldWww", icon: IconWorldWww },
  { name: "IconRouter", icon: IconRouter },
  { name: "IconDeviceSdCard", icon: IconDeviceSdCard },
  { name: "IconDeviceTv", icon: IconDeviceTv },
  { name: "IconPhoto", icon: IconPhoto },
  { name: "IconMusic", icon: IconMusic },
  { name: "IconBooks", icon: IconBooks },
  { name: "IconFile", icon: IconFile },
  { name: "IconHome", icon: IconHome },
  { name: "IconTool", icon: IconTool },
  { name: "IconApi", icon: IconApi },
  { name: "IconCode", icon: IconCode },
  { name: "IconLink", icon: IconLink },
  { name: "IconLock", icon: IconLock },
  { name: "IconKey", icon: IconKey },
  { name: "IconShieldLock", icon: IconShieldLock },
  { name: "IconMail", icon: IconMail },
];

const HOST_ICON_MAP = new Map(HOST_ICON_OPTIONS.map(({ name, icon }) => [name, icon]));

export function resolveHostIcon(icon?: string | null) {
  if (icon) {
    const tabler = HOST_ICON_MAP.get(icon);
    if (tabler) return makeTablerRenderer(tabler);
    if (EMOJI_RE.test(icon)) return makeEmojiRenderer(icon);
  }
  return makeTablerRenderer(IconServer);
}

export function validateIconInput(raw: string) {
  return validateIconWithMap(HOST_ICON_MAP, raw);
}
