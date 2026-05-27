import {
  IconDeviceMobile,
  IconDevices,
} from "@tabler/icons-react";
import type { MantineColor } from "@mantine/core";
import type { Icon as TablerIcon } from "@tabler/icons-react";
import {
  // Keep Tabler imports that were only used for backward-compat ICON_MAP entries.
  // They are intentionally not re-exported – the picker no longer surfaces them.
  IconBrandAndroid,
  IconBrandApple,
  IconCloud,
  IconCpu,
  IconDatabase,
  IconDeviceDesktop,
  IconDeviceGamepad,
  IconDeviceLaptop,
  IconDeviceTablet,
  IconDeviceTv,
  IconDeviceWatch,
  IconRouter,
  IconServer,
} from "@tabler/icons-react";
import {
  EMOJI_RE,
  makeEmojiRenderer,
  makeTablerRenderer,
  makeUrlRenderer,
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

// ─── Backward-compat map: Tabler icon names stored in the DB before the emoji
// picker migration.  Kept private — the picker no longer emits these names.
const LEGACY_ICON_MAP = new Map<string, TablerIcon>([
  ["IconDeviceMobile", IconDeviceMobile],
  ["IconDeviceLaptop", IconDeviceLaptop],
  ["IconDeviceDesktop", IconDeviceDesktop],
  ["IconServer", IconServer],
  ["IconCpu", IconCpu],
  ["IconDatabase", IconDatabase],
  ["IconRouter", IconRouter],
  ["IconDeviceTablet", IconDeviceTablet],
  ["IconDeviceTv", IconDeviceTv],
  ["IconBrandAndroid", IconBrandAndroid],
  ["IconBrandApple", IconBrandApple],
  ["IconDeviceWatch", IconDeviceWatch],
  ["IconDeviceGamepad", IconDeviceGamepad],
  ["IconCloud", IconCloud],
  ["IconDevices", IconDevices],
]);

// ─── Emoji suggestion buckets ───────────────────────────────────────────────
// Each bucket lists 12 emoji in relevance order (best match first).
// The picker renders them as a 6 × 2 grid.

const EMOJI_KEYWORD_BUCKETS: Array<[RegExp, string[]]> = [
  [
    /\b(laptop|mac\s*m\d|macbook|notebook)\b/i,
    ["💻", "🖥️", "🖨️", "⌨️", "📡", "🔌", "🎮", "📺", "⌚", "🗄️", "🤖", "🏠"],
  ],
  [
    /\b(desktop|workstation|imac|pc)\b/i,
    ["🖥️", "💻", "🖨️", "⌨️", "📡", "🔌", "🎮", "📺", "⌚", "🗄️", "🤖", "🏠"],
  ],
  [
    /\b(phone|mobile|pixel|iphone|galaxy|android)\b/i,
    ["📱", "📲", "⌚", "💻", "🖥️", "🎮", "📺", "🔌", "📡", "🗄️", "🤖", "🏠"],
  ],
  [
    /\b(server|vm|nas|node|host)\b/i,
    ["🗄️", "🖥️", "📡", "🔌", "⚙️", "💾", "💻", "🤖", "🏠", "🎮", "📺", "⌚"],
  ],
  [
    /\b(router|ap|wifi|gateway)\b/i,
    ["📡", "🔌", "🗄️", "🖥️", "⚙️", "💾", "💻", "🤖", "🏠", "🎮", "📺", "⌚"],
  ],
  [
    /\b(tablet|ipad)\b/i,
    ["📱", "💻", "⌚", "🖥️", "🎮", "📺", "🔌", "📡", "🗄️", "🤖", "🏠", "🖨️"],
  ],
  [
    /\b(tv|television)\b/i,
    ["📺", "🎮", "🖥️", "📡", "🔌", "💻", "📱", "⌚", "🗄️", "🤖", "🏠", "🖨️"],
  ],
  [
    /\b(watch)\b/i,
    ["⌚", "📱", "💻", "🖥️", "🎮", "📺", "🔌", "📡", "🗄️", "🤖", "🏠", "🖨️"],
  ],
  [
    /\b(gamepad|game|console|xbox|playstation|nintendo)\b/i,
    ["🎮", "🕹️", "📺", "🖥️", "🔌", "📡", "💻", "📱", "⌚", "🗄️", "🤖", "🏠"],
  ],
  [
    /\b(printer|scanner)\b/i,
    ["🖨️", "🖥️", "💻", "🔌", "📡", "🎮", "📺", "⌚", "🗄️", "🤖", "🏠", "📱"],
  ],
  [
    /\b(camera|cam|webcam)\b/i,
    ["📷", "🖥️", "💻", "📱", "🔌", "📡", "🎮", "📺", "⌚", "🗄️", "🤖", "🏠"],
  ],
  [
    /\b(home|smart|hub)\b/i,
    ["🏠", "📡", "🔌", "🖥️", "💻", "📱", "🎮", "📺", "⌚", "🗄️", "🤖", "🖨️"],
  ],
];

/** Default set shown when no device-name keyword matches. */
export const DEVICE_EMOJI_DEFAULT: string[] = [
  "💻", "🖥️", "📱", "🖨️", "📡", "🔌", "🎮", "📺", "⌚", "🗄️", "🤖", "🏠",
];

/**
 * Returns 12 emoji suggestions ordered by relevance to the given device name.
 * Falls back to DEVICE_EMOJI_DEFAULT when no keyword matches.
 */
export function getSuggestedEmoji(name: string): string[] {
  for (const [pattern, emojis] of EMOJI_KEYWORD_BUCKETS) {
    if (pattern.test(name)) return emojis;
  }
  return DEVICE_EMOJI_DEFAULT;
}

/**
 * Returns the single best emoji match for a device name, or "" if no keyword
 * matches.  Used by CreateDeviceModal for the auto-suggest-from-name feature.
 */
export function suggestIcon(name: string): string {
  for (const [pattern, emojis] of EMOJI_KEYWORD_BUCKETS) {
    if (pattern.test(name)) return emojis[0] ?? "";
  }
  return "";
}

// ─── Icon rendering ──────────────────────────────────────────────────────────

function isHttpsUrl(value: string) {
  return value.startsWith("https://");
}

export function resolveDeviceIcon(icon?: string | null) {
  if (icon) {
    const legacy = LEGACY_ICON_MAP.get(icon);
    if (legacy) return makeTablerRenderer(legacy);
    if (EMOJI_RE.test(icon)) return makeEmojiRenderer(icon);
    if (isHttpsUrl(icon)) return makeUrlRenderer(icon);
  }
  return makeTablerRenderer(IconDevices);
}

export function getDeviceIcon(device: {
  device_type: string;
  icon?: string | null;
}) {
  if (device.icon) {
    const legacy = LEGACY_ICON_MAP.get(device.icon);
    if (legacy) return makeTablerRenderer(legacy, { color: "var(--mantine-color-dimmed)" });
    if (EMOJI_RE.test(device.icon)) return makeEmojiRenderer(device.icon);
    if (isHttpsUrl(device.icon)) return makeUrlRenderer(device.icon);
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

// ─── Validation ──────────────────────────────────────────────────────────────

export function validateDeviceIconInput(raw: string) {
  const trimmed = raw.trim();
  if (trimmed === "") return { ok: true as const };
  // Legacy Tabler names already stored in the database.
  if (LEGACY_ICON_MAP.has(trimmed)) return { ok: true as const };
  // Standard emoji validation (via shared utility map — accepts EMOJI_RE).
  const result = validateIconWithMap(new Map(), trimmed);
  if (result.ok) return result;
  // HTTPS image URLs.
  if (isHttpsUrl(trimmed) && trimmed.length > 8) return { ok: true as const };
  return {
    ok: false as const,
    reason: "Enter an emoji or an https:// image URL.",
  };
}
