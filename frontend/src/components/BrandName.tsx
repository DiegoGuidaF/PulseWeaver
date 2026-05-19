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
        <span style={{ color: "var(--mantine-color-orange-4)" }}>Pulse</span>Weaver
      </span>
    </span>
  );
}
