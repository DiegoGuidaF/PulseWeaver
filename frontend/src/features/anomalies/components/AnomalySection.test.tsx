import { describe, expect, it } from "vitest";
import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";
import { TEST_TIMEOUTS } from "@/test/constants";
import { anomalyHandlers, endpoints, responses } from "@/test/mocks/handlers";
import { createMockAnomaly } from "@/test/mocks/data";
import { AnomalySeverity } from "@/lib/api";
import { AnomalySection } from "./AnomalySection";

describe("AnomalySection", () => {
    it("renders rows from mocked open anomalies", async () => {
        server.use(
            anomalyHandlers.list([
                createMockAnomaly({
                    id: 1,
                    severity: AnomalySeverity.WARNING,
                    summary: "48 denials in an hour vs a typical 3.",
                }),
            ]),
        );
        renderWithProviders(<AnomalySection />);

        await waitFor(
            () => expect(screen.getByText("48 denials in an hour vs a typical 3.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByText("Deny spike")).toBeInTheDocument();
    });

    it("hides info-severity anomalies", async () => {
        server.use(
            anomalyHandlers.list([
                createMockAnomaly({ id: 1, severity: AnomalySeverity.INFO, summary: "Info-only finding." }),
                createMockAnomaly({ id: 2, severity: AnomalySeverity.WARNING, summary: "Warning finding." }),
            ]),
        );
        renderWithProviders(<AnomalySection />);

        await waitFor(
            () => expect(screen.getByText("Warning finding.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.queryByText("Info-only finding.")).not.toBeInTheDocument();
    });

    it("caps rows at 5 and links to the full count", async () => {
        const many = Array.from({ length: 7 }, (_, i) =>
            createMockAnomaly({
                id: i + 1,
                severity: AnomalySeverity.CRITICAL,
                summary: `Finding number ${i + 1}.`,
            }),
        );
        server.use(anomalyHandlers.list(many));
        renderWithProviders(<AnomalySection />);

        await waitFor(
            () => expect(screen.getByText("Finding number 1.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.queryByText("Finding number 6.")).not.toBeInTheDocument();
        expect(screen.getByText("View all 7 →")).toBeInTheDocument();
    });

    it("shows a calm empty state when there is no unusual activity", async () => {
        server.use(anomalyHandlers.list([]));
        renderWithProviders(<AnomalySection />);

        await waitFor(
            () => expect(screen.getByText("No unusual activity")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows an error state with retry on failure", async () => {
        server.use(http.get(endpoints.anomalies, () => responses.serverError()));
        renderWithProviders(<AnomalySection />);

        await waitFor(
            () => expect(screen.getByText("Failed to load anomalies")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument();
    });

    it("acknowledging a row removes it once the cache is invalidated", async () => {
        let anomalies = [createMockAnomaly({ id: 5, severity: AnomalySeverity.CRITICAL, summary: "Ack me." })];
        server.use(
            http.get(endpoints.anomalies, () => HttpResponse.json({ anomalies })),
            http.post(endpoints.anomalyAcknowledge, () => {
                anomalies = [];
                return responses.noContent();
            }),
        );

        const user = setupUser();
        renderWithProviders(<AnomalySection />);

        await waitFor(
            () => expect(screen.getByText("Ack me.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByRole("button", { name: /acknowledge/i }));

        await waitFor(
            () => expect(screen.queryByText("Ack me.")).not.toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });
});
