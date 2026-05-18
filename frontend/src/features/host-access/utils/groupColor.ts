export const GROUP_COLOR_PALETTE = [
  "#4C6EF5",
  "#7950F2",
  "#F06595",
  "#74C0FC",
  "#63E6BE",
  "#69DB7C",
  "#A9E34B",
  "#4DABF7",
  "#66D9E8",
  "#94D82D",
  "#F783AC",
  "#748FFC",
] as const;

export type GroupColor = (typeof GROUP_COLOR_PALETTE)[number];

export function getLeastUsedColor(existingColors: string[]): string {
  const counts = new Map<string, number>(GROUP_COLOR_PALETTE.map((c) => [c, 0]));
  for (const c of existingColors) {
    if (counts.has(c)) counts.set(c, (counts.get(c) ?? 0) + 1);
  }
  return [...counts.entries()].reduce((a, b) => (b[1] < a[1] ? b : a))[0];
}

export function getContrastColor(hex: string): "#000000" | "#ffffff" {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  return luminance > 0.5 ? "#000000" : "#ffffff";
}
