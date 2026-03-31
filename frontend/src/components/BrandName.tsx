import "@fontsource/space-grotesk/700.css";

interface BrandNameProps {
  size?: string | number;
  style?: React.CSSProperties;
  className?: string;
}

export function BrandName({ size = "1.4rem", style, className }: BrandNameProps) {
  return (
    <span
      style={{
        fontFamily: "'Space Grotesk', sans-serif",
        fontWeight: 700,
        fontSize: size,
        letterSpacing: "-0.02em",
        lineHeight: 1,
        ...style,
      }}
      className={className}
    >
      <span style={{ color: "var(--mantine-color-orange-4)" }}>Pulse</span>Weaver
    </span>
  );
}
