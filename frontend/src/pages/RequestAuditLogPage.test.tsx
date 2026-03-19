import { describe, expect, it } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http } from "msw";
import { server } from "@/test/setup";
import { renderWithProviders } from "@/test/utils";
import { RequestAuditLogPage } from "./RequestAuditLogPage";
import {
    createMockRequestAuditLogRow,
    createMockRequestAuditLogResponse,
} from "@/test/mocks/data";
import { endpoints, requestAuditLogHandlers, responses } from "@/test/mocks/handlers";
import { TEST_TIMEOUTS } from "@/test/constants";

// Pre-set date range so the table's useEffect does not trigger
// a second query (avoiding the double-render that causes test flakiness).
const BASE_ENTRY =
    "/request-audit-log?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z";

describe("RequestAuditLogPage", () => {
    it("renders table with mock rows", async () => {
        const row = createMockRequestAuditLogRow({
            client_ip: "203.0.113.42",
            target_host: "example.com",
            outcome: true,
        });
        server.use(
            requestAuditLogHandlers.list(
                createMockRequestAuditLogResponse({ rows: [row], total: 1 }),
            ),
        );

        renderWithProviders(<RequestAuditLogPage />, {
            initialEntries: [BASE_ENTRY],
        });

        await waitFor(
            () => {
                expect(screen.getByText("203.0.113.42")).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("example.com")).toBeInTheDocument();
        // "Allow" appears as the outcome badge in the table row
        expect(screen.getAllByText("Allow").length).toBeGreaterThan(0);
        expect(screen.getByText("1 result")).toBeInTheDocument();
    });

    it("shows empty state when rows is empty", async () => {
        server.use(
            requestAuditLogHandlers.list(
                createMockRequestAuditLogResponse({ rows: [], total: 0 }),
            ),
        );

        renderWithProviders(<RequestAuditLogPage />, {
            initialEntries: [BASE_ENTRY],
        });

        await waitFor(
            () => {
                expect(
                    screen.getByText("No matching log entries."),
                ).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows error state when API returns 500", async () => {
        server.use(
            http.get(endpoints.requestAuditLog, () => responses.serverError()),
        );

        renderWithProviders(<RequestAuditLogPage />, {
            initialEntries: [BASE_ENTRY],
        });

        await waitFor(
            () => {
                expect(screen.getByText("Failed to load")).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows error state when API returns 403", async () => {
        server.use(
            http.get(endpoints.requestAuditLog, () =>
                responses.forbidden({ message: "Forbidden - admin credentials required" }),
            ),
        );

        renderWithProviders(<RequestAuditLogPage />, {
            initialEntries: [BASE_ENTRY],
        });

        await waitFor(
            () => {
                expect(screen.getByText("Failed to load")).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("row click opens detail drawer with row data", async () => {
        const user = userEvent.setup();
        const row = createMockRequestAuditLogRow({
            id: 42,
            client_ip: "10.0.0.1",
            outcome: false,
            deny_reason: "invalid_token",
            target_host: "secure.example.com",
        });
        server.use(
            requestAuditLogHandlers.list(
                createMockRequestAuditLogResponse({ rows: [row], total: 1 }),
            ),
        );

        renderWithProviders(<RequestAuditLogPage />, {
            initialEntries: [BASE_ENTRY],
        });

        await waitFor(
            () => {
                expect(screen.getByText("10.0.0.1")).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByRole("button", { name: "View details" }));

        await waitFor(
            () => {
                expect(screen.getByText("Request Detail")).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        // "Invalid token" appears in both table row and detail drawer
        expect(screen.getAllByText("Invalid token").length).toBeGreaterThan(0);
        expect(screen.getAllByText("secure.example.com").length).toBeGreaterThan(0);
    });

    it("outcome column filter opens and shows options", async () => {
        const user = userEvent.setup();

        renderWithProviders(<RequestAuditLogPage />, {
            initialEntries: [BASE_ENTRY],
        });

        // Wait for the table to render (column headers should be present)
        await waitFor(
            () => {
                expect(screen.getByRole("columnheader", { name: /outcome/i })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        // Open the column filter popover
        const outcomeHeader = screen.getByRole("columnheader", { name: /outcome/i });
        const filterButton = within(outcomeHeader).getByRole("button");
        await user.click(filterButton);

        // The SegmentedControl with "Deny" should now be visible
        await waitFor(
            () => {
                expect(screen.getByText("Deny")).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByText("Deny"));

        // Verify the component doesn't crash and "Deny" remains visible
        expect(screen.getByText("Deny")).toBeInTheDocument();
    });
});
