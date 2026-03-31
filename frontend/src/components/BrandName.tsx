interface BrandNameProps {
  style?: React.CSSProperties;
  className?: string;
}

export function BrandName({ style, className }: BrandNameProps) {
  return (
    <span style={style} className={className}>
      <span style={{ color: "var(--mantine-color-orange-4)" }}>Pulse</span>Weaver
    </span>
  );
}
