import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { server } from "@/test/setup";
import { renderWithProviders } from "@/test/utils";
import { AuthProvider } from "@/features/auth/AuthContext";
import { TEST_TIMEOUTS } from "@/test/constants";
import { anomalyHandlers } from "@/test/mocks/handlers";
import { createMockAnomaly } from "@/test/mocks/data";
import { AppShell } from "./AppShell";

function renderAppShell() {
    return renderWithProviders(
        <AuthProvider>
            <AppShell>{null}</AppShell>
        </AuthProvider>,
    );
}

describe("AppShell nav badge", () => {
    it("hides the Anomalies badge when there are no open anomalies", async () => {
        server.use(anomalyHandlers.list([]));
        renderAppShell();

        await waitFor(
            () => expect(screen.getByText("Anomalies")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.queryByText(/^\d+$/)).not.toBeInTheDocument();
    });

    it("shows the open anomaly count on the Anomalies nav item", async () => {
        server.use(anomalyHandlers.list([createMockAnomaly({ id: 1 }), createMockAnomaly({ id: 2 })]));
        renderAppShell();

        await waitFor(
            () => expect(screen.getByText("2")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });
});
