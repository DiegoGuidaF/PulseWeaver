import { useCallback } from "react";
import { Stack, Grid } from "@mantine/core";
import { useNavigate } from "react-router-dom";
import { ROUTES } from "@/lib/routes";
import { useCountryStats } from "../hooks/useCountryStats";
import { useCountryLookup } from "../hooks/useCountryLookup";
import { useMapColorScale } from "../hooks/useMapColorScale";
import { AccessMap } from "./AccessMap";
import { TopCountriesTable } from "./TopCountriesTable";
import { ErrorState } from "@/components/ErrorState";

interface CountryStatsSectionProps {
    from?: string;
    to?: string;
}

export function CountryStatsSection({ from, to }: CountryStatsSectionProps) {
    const navigate = useNavigate();

    const { data, isLoading, error, refetch } = useCountryStats(from, to);
    const lookup = useCountryLookup(data);
    const colorFn = useMapColorScale(data);

    const handleCountryClick = useCallback(
        (code: string) => navigate(`${ROUTES.accessLog}?country_code=${code}`),
        [navigate],
    );

    if (error) {
        return <ErrorState error={error} title="Failed to load country stats" onRetry={() => refetch()} />;
    }

    // Hide the entire section when geo data is unavailable (GeoIP disabled or no enriched records).
    if (!isLoading && (data?.length ?? 0) === 0) return null;

    return (
        <Stack gap="xs">
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
                        onCountryClick={handleCountryClick}
                    />
                </Grid.Col>
            </Grid>
        </Stack>
    );
}
