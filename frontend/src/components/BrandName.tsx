import "@fontsource/space-grotesk/700.css";
import { useComputedColorScheme } from "@mantine/core";
import markLight from "@/assets/brand/mark-light.svg";
import markDark from "@/assets/brand/mark-dark.svg";

interface BrandNameProps {
  size?: string | number;
  style?: React.CSSProperties;
  className?: string;
}

export function BrandName({ size = "1.4rem", style, className }: BrandNameProps) {
  const colorScheme = useComputedColorScheme("light");
  const mark = colorScheme === "dark" ? markDark : markLight;
  // PW-64: orange-4 (#FFA94D) fails contrast on light backgrounds. The wordmark is
  // large bold text (≥18.66px/700 → AA needs only 3:1), so a darker-but-still-vivid
  // orange-7 (#f76707, ~3.6:1) keeps the brand amber while passing. Dark keeps orange-4.
  const pulseColor =
    colorScheme === "dark"
      ? "var(--mantine-color-orange-4)"
      : "var(--mantine-color-orange-7)";
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: "0.4em",
        fontFamily: "'Space Grotesk', sans-serif",
        fontWeight: 700,
        fontSize: size,
        letterSpacing: "-0.02em",
        lineHeight: 1,
        ...style,
      }}
      className={className}
    >
      <img src={mark} alt="" style={{ height: "1.3em", width: "auto" }} />
      <span>
        <span style={{ color: pulseColor }}>Pulse</span>Weaver
      </span>
    </span>
  );
}
