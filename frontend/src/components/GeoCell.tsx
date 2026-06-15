import { Group, Stack, Text, Tooltip } from "@mantine/core";
import { countryFlagEmoji } from "@/lib/countryFlag";
import type { GeoInfo } from "@/lib/api";

interface GeoCellProps {
  geo?: GeoInfo | null;
  /** Show the ASN organization on a dimmed secondary line instead of only in the tooltip. */
  showAsn?: boolean;
  size?: "xs" | "sm";
}

/**
 * Renders GeoIP metadata for an IP: country flag + code, with country name and
 * ASN organization in a tooltip. Renders nothing when geo is absent (private or
 * unresolvable IP, or GeoIP disabled) or carries no displayable value.
 */
export function GeoCell({ geo, showAsn = false, size = "sm" }: GeoCellProps) {
  if (!geo) return null;

  const flag = geo.country_code ? countryFlagEmoji(geo.country_code) : "";
  const label = geo.country_code ?? "";
  const asnOrg = geo.asn_org ?? "";

  // Nothing meaningful to show.
  if (!label && !asnOrg) return null;

  const tooltipParts = [
    [flag, geo.country_name ?? geo.country_code, geo.country_code ? `(${geo.country_code})` : ""]
      .filter(Boolean)
      .join(" "),
    asnOrg && `AS${geo.asn ?? ""} ${asnOrg}`.trim(),
  ].filter(Boolean);

  const primary = (
    <Text size={size} component="span">
      {flag && `${flag} `}
      {label || asnOrg}
    </Text>
  );

  const content =
    showAsn && asnOrg && label ? (
      <Stack gap={0}>
        {primary}
        <Text size="xs" c="dimmed">
          {asnOrg}
        </Text>
      </Stack>
    ) : (
      primary
    );

  return (
    <Tooltip label={tooltipParts.join(" · ")} withArrow multiline>
      <Group gap={4} wrap="nowrap" display="inline-flex">
        {content}
      </Group>
    </Tooltip>
  );
}
