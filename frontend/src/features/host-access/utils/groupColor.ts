import type { MantineColor } from "@mantine/core";

const GROUP_COLORS: MantineColor[] = [
  "indigo",
  "violet",
  "teal",
  "cyan",
  "grape",
  "pink",
  "lime",
  "green",
];

export function groupColor(name: string): MantineColor {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = ((h * 31) + name.charCodeAt(i)) >>> 0;
  return GROUP_COLORS[h % GROUP_COLORS.length];
}
