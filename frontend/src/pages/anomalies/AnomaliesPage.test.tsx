import { describe, expect, it } from "vitest";
import { http, HttpResponse } from "msw";
import { screen, waitFor, within } from "@testing-library/react";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";
import { TEST_TIMEOUTS } from "@/test/constants";
import { anomalyHandlers, endpoints, responses } from "@/test/mocks/handlers";
import { createMockAnomaly } from "@/test/mocks/data";
import { AnomalyKind, AnomalySeverity, AnomalyStatus } from "@/lib/api";
import { AnomaliesPage } from "./AnomaliesPage";

describe("AnomaliesPage", () => {
    it("defaults the status filter to Open and requests open anomalies", async () => {
        let capturedStatus: string | null = null;
        server.use(
            http.get(endpoints.anomalies, ({ request }) => {
                capturedStatus = new URL(request.url).searchParams.get("status");
                return HttpResponse.json({ anomalies: [createMockAnomaly({ summary: "Open finding." })] });
            }),
        );

        renderWithProviders(<AnomaliesPage />);

        await waitFor(
            () => expect(screen.getByText("Open finding.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(capturedStatus).toBe("open");
    });

    it("shows an empty state when no anomalies match the filters", async () => {
        server.use(anomalyHandlers.list([]));
        renderWithProviders(<AnomaliesPage />);

        await waitFor(
            () => expect(screen.getByText("No anomalies match these filters")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows an error state with retry on failure", async () => {
        server.use(http.get(endpoints.anomalies, () => responses.serverError()));
        renderWithProviders(<AnomaliesPage />);

        await waitFor(
            () => expect(screen.getByText("Failed to load anomalies")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("expanding a row shows its raw evidence entries", async () => {
        server.use(
            anomalyHandlers.list([
                createMockAnomaly({
                    summary: "Expandable finding.",
                    evidence: { deny_count: 12, target_hosts: ["a.example.com", "b.example.com"] },
                }),
            ]),
        );
        const user = setupUser();
        renderWithProviders(<AnomaliesPage />);

        await waitFor(
            () => expect(screen.getByText("Expandable finding.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByRole("button", { name: /show evidence/i }));

        expect(screen.getByText("deny_count")).toBeInTheDocument();
        expect(screen.getByText("12")).toBeInTheDocument();
        expect(screen.getByText("a.example.com, b.example.com")).toBeInTheDocument();
    });

    it("changing the kind filter re-queries the SDK with the selected kind", async () => {
        let capturedKinds: string[] = [];
        server.use(
            http.get(endpoints.anomalies, ({ request }) => {
                capturedKinds = new URL(request.url).searchParams.getAll("kind");
                return HttpResponse.json({ anomalies: [] });
            }),
        );

        const user = setupUser();
        renderWithProviders(<AnomaliesPage />);

        await waitFor(
            () => expect(screen.getByText("No anomalies match these filters")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        const kindFilter = screen.getByRole("combobox", { name: /filter by kind/i });
        await user.click(kindFilter);
        await user.click(await screen.findByText("Host probing"));

        await waitFor(
            () => expect(capturedKinds).toEqual([AnomalyKind.HOST_PROBING]),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("acknowledge is hidden for already-acknowledged anomalies", async () => {
        server.use(
            anomalyHandlers.list([
                createMockAnomaly({ status: AnomalyStatus.ACKNOWLEDGED, summary: "Already handled." }),
            ]),
        );
        renderWithProviders(<AnomaliesPage />);

        await waitFor(
            () => expect(screen.getByText("Already handled.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.queryByRole("button", { name: /^acknowledge$/i })).not.toBeInTheDocument();
    });

    it("renders severity and summary for each row", async () => {
        server.use(
            anomalyHandlers.list([
                createMockAnomaly({ severity: AnomalySeverity.CRITICAL, summary: "Critical finding." }),
            ]),
        );
        renderWithProviders(<AnomaliesPage />);

        await waitFor(
            () => expect(screen.getByText("Critical finding.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        const row = screen.getByText("Critical finding.").closest(".mantine-Card-root") as HTMLElement;
        expect(within(row).getByText("Critical")).toBeInTheDocument();
    });
});
