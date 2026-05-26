import React from "react";
import type { Icon as TablerIcon } from "@tabler/icons-react";

export type IconRenderer = (props: { size: number; style?: React.CSSProperties }) => React.ReactNode;
export type IconValidation = { ok: true } | { ok: false; reason: string };

// A single grapheme cluster whose first codepoint is an Extended_Pictographic.
// Allows VS16 (U+FE0F) and ZWJ sequences (U+200D) — covers emoji like 👨‍👩‍👧.
export const EMOJI_RE =
  /^\p{Extended_Pictographic}(️|‍\p{Extended_Pictographic}️?)*$/u;

export function makeTablerRenderer(icon: TablerIcon, colorStyle?: React.CSSProperties): IconRenderer {
  return ({ size, style }) =>
    React.createElement(icon, { size, stroke: 1.5, style: { ...colorStyle, ...style } });
}

export function makeEmojiRenderer(value: string): IconRenderer {
  return ({ size, style }) =>
    React.createElement("span", { style: { fontSize: size - 2, lineHeight: 1, ...style } }, value);
}

export function validateIconWithMap(map: Map<string, unknown>, raw: string): IconValidation {
  const trimmed = raw.trim();
  if (trimmed === "") return { ok: true };
  if (map.has(trimmed)) return { ok: true };
  if (EMOJI_RE.test(trimmed)) return { ok: true };
  return {
    ok: false,
    reason: "Enter a single emoji or pick an icon from the suggestions.",
  };
}
