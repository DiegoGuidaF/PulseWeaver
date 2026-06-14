import { Stack } from "@mantine/core";
import { PageToolbar } from "@/components/PageToolbar";
import { DashboardView } from "@/features/dashboard/components/DashboardView";

export function TrafficDashboardPage() {
    return (
        <Stack gap="xl">
            <h1 style={{ position: "absolute", width: 1, height: 1, padding: 0, margin: -1, overflow: "hidden", clip: "rect(0,0,0,0)", whiteSpace: "nowrap", border: 0 }}>Dashboard</h1>
            <PageToolbar subtitle="Security posture and traffic" />
            <DashboardView />
        </Stack>
    );
}
