import { Stack } from "@mantine/core";
import { PageToolbar } from "@/components/PageToolbar";
import { AnomaliesList } from "@/features/anomalies/components/AnomaliesList";

export function AnomaliesPage() {
    return (
        <Stack gap="xl">
            <h1
                style={{
                    position: "absolute",
                    width: 1,
                    height: 1,
                    padding: 0,
                    margin: -1,
                    overflow: "hidden",
                    clip: "rect(0,0,0,0)",
                    whiteSpace: "nowrap",
                    border: 0,
                }}
            >
                Anomalies
            </h1>
            <PageToolbar subtitle="Detected anomalies" />
            <AnomaliesList />
        </Stack>
    );
}
