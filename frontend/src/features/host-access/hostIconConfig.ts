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

export function getHostIcon(icon?: string | null): TablerIcon {
  if (icon) {
    const resolved = HOST_ICON_MAP.get(icon);
    if (resolved) return resolved;
  }
  return IconServer;
}

export type ResolvedHostIcon =
  | { kind: "tabler"; icon: TablerIcon }
  | { kind: "emoji"; value: string };

// A single grapheme cluster whose first codepoint is an Extended_Pictographic.
// Allows VS16 (U+FE0F) and ZWJ sequences (U+200D) — covers emoji like 👨‍👩‍👧.
const EMOJI_RE =
  /^\p{Extended_Pictographic}(️|‍\p{Extended_Pictographic}️?)*$/u;

export function resolveHostIcon(icon: string | null | undefined): ResolvedHostIcon {
  if (icon) {
    const tabler = HOST_ICON_MAP.get(icon);
    if (tabler) return { kind: "tabler", icon: tabler };
    if (EMOJI_RE.test(icon)) return { kind: "emoji", value: icon };
  }
  return { kind: "tabler", icon: IconServer };
}

export type IconValidation = { ok: true } | { ok: false; reason: string };

export function validateIconInput(raw: string): IconValidation {
  const trimmed = raw.trim();
  if (trimmed === "") return { ok: true };
  if (HOST_ICON_MAP.has(trimmed)) return { ok: true };
  if (EMOJI_RE.test(trimmed)) return { ok: true };
  return {
    ok: false,
    reason: "Enter a single emoji or pick an icon from the suggestions.",
  };
}
