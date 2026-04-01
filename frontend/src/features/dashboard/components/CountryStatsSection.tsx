import { useState, useCallback } from "react";
import { Stack, Grid, Group, SegmentedControl } from "@mantine/core";
import { useNavigate } from "react-router-dom";
import { useCountryStats } from "../hooks/useCountryStats";
import { useCountryLookup } from "../hooks/useCountryLookup";
import { useMapColorScale } from "../hooks/useMapColorScale";
import { AccessMap } from "./AccessMap";
import { TopCountriesTable } from "./TopCountriesTable";

type Metric = "denied" | "total";

interface CountryStatsSectionProps {
    from?: string;
    to?: string;
}

export function CountryStatsSection({ from, to }: CountryStatsSectionProps) {
    const [metric, setMetric] = useState<Metric>("denied");
    const navigate = useNavigate();

    const { data, isLoading } = useCountryStats(from, to);
    const lookup = useCountryLookup(data);
    const colorFn = useMapColorScale(data, metric);

    const handleCountryClick = useCallback(
        (code: string) => navigate(`/access-log?country_code=${code}`),
        [navigate],
    );

    return (
        <Stack gap="xs">
            <Group justify="flex-end">
                <SegmentedControl
                    size="xs"
                    value={metric}
                    onChange={(v) => setMetric(v as Metric)}
                    data={[
                        { value: "denied", label: "Denied" },
                        { value: "total", label: "Total" },
                    ]}
                />
            </Group>
            <Grid>
                <Grid.Col span={{ base: 12, md: 8 }}>
                    <AccessMap
                        data={data}
                        isLoading={isLoading}
                        colorFn={colorFn}
                        lookup={lookup}
                        onCountryClick={handleCountryClick}
                    />
                </Grid.Col>
                <Grid.Col span={{ base: 12, md: 4 }}>
                    <TopCountriesTable
                        data={data}
                        isLoading={isLoading}
                        metric={metric}
                        onCountryClick={handleCountryClick}
                    />
                </Grid.Col>
            </Grid>
        </Stack>
    );
}
