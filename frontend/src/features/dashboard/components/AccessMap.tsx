import { useState, useCallback, useMemo } from "react";
import { Paper, Text, Skeleton, Box, Group } from "@mantine/core";
import { useElementSize } from "@mantine/hooks";
import { IconMap } from "@tabler/icons-react";
import { geoNaturalEarth1, geoPath } from "d3-geo";
import { feature } from "topojson-client";
import type { Feature, FeatureCollection, Geometry } from "geojson";
import type { Topology } from "topojson-specification";
import worldData from "world-atlas/countries-110m.json";
import { numericToAlpha2 } from "@/lib/countryCodeMap";
import { countryFlagEmoji } from "@/lib/countryFlag";
import { EmptyState } from "@/components/EmptyState";
import type { AccessLogCountryStats } from "@/lib/api/types.gen";

// Convert TopoJSON → GeoJSON once at module level (no re-parsing per render)
const worldTopo = worldData as unknown as Topology;
const countries = feature(
    worldTopo,
    worldTopo.objects.countries,
) as FeatureCollection<Geometry>;

// Bounding box that excludes Antarctica: clip at 60°S and 83°N.
// geoNaturalEarth1().fitExtent() will scale/translate the projection so these
// bounds fill the SVG area; anything outside (Antarctica) is clipped by viewBox.
const CLIP_BOUNDS: Feature = {
    type: "Feature",
    properties: null,
    geometry: {
        type: "Polygon",
        coordinates: [
            [
                [-179.9, -60],
                [179.9, -60],
                [179.9, 83],
                [-179.9, 83],
                [-179.9, -60],
            ],
        ],
    },
};

const MAP_HEIGHT = 420;

interface AccessMapProps {
    data: AccessLogCountryStats[] | undefined;
    isLoading: boolean;
    colorFn: (
        countryCode: string,
        lookup: Map<string, AccessLogCountryStats>,
    ) => string;
    lookup: Map<string, AccessLogCountryStats>;
    onCountryClick: (code: string) => void;
}

interface TooltipState {
    stats: AccessLogCountryStats;
    x: number;
    y: number;
}

export function AccessMap({
    data,
    isLoading,
    colorFn,
    lookup,
    onCountryClick,
}: AccessMapProps) {
    const { ref, width } = useElementSize();
    const [tooltip, setTooltip] = useState<TooltipState | null>(null);

    // Fall back to 800 until the container is measured (also safe in jsdom tests)
    const w = Math.max(width, 400);

    const projection = useMemo(
        () =>
            geoNaturalEarth1().fitExtent(
                [
                    [2, 2],
                    [w - 2, MAP_HEIGHT - 2],
                ],
                CLIP_BOUNDS,
            ),
        [w],
    );

    const pathGen = useMemo(
        () => geoPath().projection(projection),
        [projection],
    );

    const handleMouseMove = useCallback(
        (alpha2: string, e: React.MouseEvent) => {
            const stats = lookup.get(alpha2);
            if (stats) setTooltip({ stats, x: e.clientX, y: e.clientY });
            else setTooltip(null);
        },
        [lookup],
    );

    const handleMouseLeave = useCallback(() => setTooltip(null), []);

    const handleClick = useCallback(
        (alpha2: string) => {
            if (lookup.has(alpha2)) onCountryClick(alpha2);
        },
        [lookup, onCountryClick],
    );

    return (
        <Paper withBorder p="md" radius="md">
            <Group justify="space-between" mb="md">
                <Text fw={500}>Access Map</Text>
                <Text size="xs" c="dimmed">
                    Shaded by deny rate
                </Text>
            </Group>
            {isLoading ? (
                <Skeleton h={MAP_HEIGHT} />
            ) : !data || data.length === 0 ? (
                <EmptyState
                    icon={IconMap}
                    title="No geographic data in this period"
                />
            ) : (
                <Box pos="relative" ref={ref}>
                    <svg
                        viewBox={`0 0 ${w} ${MAP_HEIGHT}`}
                        role="img"
                        style={{ width: "100%", height: "auto", display: "block" }}
                        aria-label="World access map — geographic distribution of requests. See the Top Countries table for detailed data."
                    >
                        {countries.features.map((geo) => {
                            const rawId = geo.id;
                            if (rawId == null) return null;
                            const numericId = String(rawId).padStart(3, "0");
                            const alpha2 = numericToAlpha2.get(numericId) ?? "";
                            const hasData = lookup.has(alpha2);
                            const fill = colorFn(alpha2, lookup);
                            return (
                                <path
                                    key={String(rawId)}
                                    d={pathGen(geo) ?? ""}
                                    fill={fill}
                                    stroke="var(--mantine-color-dark-7)"
                                    strokeWidth={0.5}
                                    style={{
                                        outline: "none",
                                        cursor: hasData ? "pointer" : "default",
                                    }}
                                    onMouseMove={(e) =>
                                        handleMouseMove(alpha2, e)
                                    }
                                    onMouseLeave={handleMouseLeave}
                                    onClick={() => handleClick(alpha2)}
                                />
                            );
                        })}
                    </svg>

                    {tooltip && (
                        <Paper
                            shadow="md"
                            p="xs"
                            radius="sm"
                            withBorder
                            style={{
                                position: "fixed",
                                left: tooltip.x + 12,
                                top: tooltip.y - 12,
                                pointerEvents: "none",
                                zIndex: 1000,
                            }}
                        >
                            <Text size="sm" fw={500}>
                                {countryFlagEmoji(tooltip.stats.country_code)}{" "}
                                {tooltip.stats.country_name ??
                                    tooltip.stats.country_code}
                            </Text>
                            <Text size="xs" c="dimmed">
                                Total: {tooltip.stats.total.toLocaleString()} |
                                Denied:{" "}
                                {tooltip.stats.denied.toLocaleString()} |
                                Allowed:{" "}
                                {tooltip.stats.allowed.toLocaleString()}
                            </Text>
                        </Paper>
                    )}
                </Box>
            )}
        </Paper>
    );
}
