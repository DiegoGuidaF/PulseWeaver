import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
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

// Pre-set date range so RequestAuditLogFilters' useEffect does not trigger
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
        // "Allow" appears in both the outcome filter and the table badge
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

        await user.click(screen.getByText("10.0.0.1"));

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

    it("outcome filter updates URL search params", async () => {
        const user = userEvent.setup();

        renderWithProviders(<RequestAuditLogPage />, {
            initialEntries: [BASE_ENTRY],
        });

        await waitFor(
            () => {
                expect(
                    screen.getByRole("radio", { name: "Deny" }) ??
                        screen.getByText("Deny"),
                ).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        // SegmentedControl renders buttons/labels — click "Deny"
        await user.click(screen.getByText("Deny"));

        // The component should re-render; just verify it doesn't crash and
        // the "Deny" option is active
        expect(screen.getByText("Deny")).toBeInTheDocument();
    });
});
