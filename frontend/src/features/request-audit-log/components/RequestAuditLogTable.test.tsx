import { describe, expect, it } from "vitest";
import { screen, waitFor, within, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http } from "msw";
import { server } from "@/test/setup";
import { renderWithProviders } from "@/test/utils";
import { RequestAuditLogPage } from "@/pages/RequestAuditLogPage";
import {
    createMockRequestAuditLogRow,
    createMockRequestAuditLogResponse,
} from "@/test/mocks/data";
import { endpoints, requestAuditLogHandlers, responses } from "@/test/mocks/handlers";
import { TEST_TIMEOUTS } from "@/test/constants";

// Pre-set date range so the component starts with a bounded time window.
const BASE_ENTRY =
    "/request-audit-log?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z";

// ─── Helpers ─────────────────────────────────────────────────────────────────

function renderTable(initialEntries = [BASE_ENTRY]) {
    return renderWithProviders(<RequestAuditLogPage />, {
        initialEntries,
    });
}

/** Returns the filter icon button inside a column header by column title text. */
function getFilterButton(columnTitle: string | RegExp) {
    const header = screen
        .getAllByRole("columnheader")
        .find((h) => (typeof columnTitle === "string"
            ? h.textContent?.includes(columnTitle)
            : columnTitle.test(h.textContent ?? "")));
    if (!header) throw new Error(`Column header "${columnTitle}" not found`);
    return within(header).getByRole("button");
}

// ─── Basic rendering ─────────────────────────────────────────────────────────

describe("RequestAuditLogTable", () => {
    it("renders table rows from API response", async () => {
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

        renderTable();

        await waitFor(
            () => expect(screen.getByText("203.0.113.42")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByText("example.com")).toBeInTheDocument();
        expect(screen.getAllByText("Allow").length).toBeGreaterThan(0);
        expect(screen.getByText("1 result")).toBeInTheDocument();
    });

    it("shows no-records message while keeping column headers visible", async () => {
        server.use(
            requestAuditLogHandlers.list(
                createMockRequestAuditLogResponse({ rows: [], total: 0 }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        // Column headers must remain visible so filters can still be changed
        expect(screen.getByRole("columnheader", { name: /time/i })).toBeInTheDocument();
        expect(
            screen.getAllByRole("columnheader").find((h) => h.textContent?.includes("IP")),
        ).toBeDefined();
        expect(screen.getByRole("columnheader", { name: /outcome/i })).toBeInTheDocument();
    });

    it("shows error alert when API returns 500", async () => {
        server.use(http.get(endpoints.requestAuditLog, () => responses.serverError()));

        renderTable();

        await waitFor(
            () => expect(screen.getByText("Failed to load")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows error alert when API returns 403", async () => {
        server.use(
            http.get(endpoints.requestAuditLog, () =>
                responses.forbidden({ message: "Forbidden - admin credentials required" }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("Failed to load")).toBeInTheDocument(),
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

        renderTable();

        await waitFor(
            () => expect(screen.getByText("10.0.0.1")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByRole("button", { name: "View details" }));

        await waitFor(
            () => expect(screen.getByText("Request Detail")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getAllByText("Invalid token").length).toBeGreaterThan(0);
        expect(screen.getAllByText("secure.example.com").length).toBeGreaterThan(0);
    });

    // ─── IP filter ─────────────────────────────────────────────────────────────

    describe("IP filter", () => {
        it("opens when the filter icon is clicked", async () => {
            const user = userEvent.setup();
            server.use(
                requestAuditLogHandlers.list(
                    createMockRequestAuditLogResponse({ rows: [], total: 0 }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("IP"));

            expect(
                await screen.findByPlaceholderText("Filter by IP"),
            ).toBeInTheDocument();
        });

        it("closes when the filter icon is clicked again (toggle)", async () => {
            const user = userEvent.setup();
            server.use(
                requestAuditLogHandlers.list(
                    createMockRequestAuditLogResponse({ rows: [], total: 0 }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            const filterBtn = getFilterButton("IP");

            // Open via userEvent (full pointer simulation)
            await user.click(filterBtn);
            expect(await screen.findByPlaceholderText("Filter by IP")).toBeInTheDocument();

            // Close via fireEvent.click — avoids the mousedown-triggered click-outside
            // that would otherwise close+reopen the popover before toggle() fires.
            fireEvent.click(filterBtn);
            await waitFor(() =>
                expect(screen.queryByPlaceholderText("Filter by IP")).not.toBeInTheDocument(),
            );
        });

        it("retains all typed characters without resetting", async () => {
            const user = userEvent.setup();
            server.use(
                requestAuditLogHandlers.list(
                    createMockRequestAuditLogResponse({ rows: [], total: 0 }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("IP"));
            const input = await screen.findByPlaceholderText("Filter by IP");
            await user.type(input, "192.168.1");

            expect(input).toHaveValue("192.168.1");
        });

        it("activates filter indicator only after debounce, not on every keystroke", async () => {
            const user = userEvent.setup();
            server.use(
                requestAuditLogHandlers.list(
                    createMockRequestAuditLogResponse({ rows: [], total: 0 }),
                ),
            );

            const { container } = renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("IP"));
            const input = await screen.findByPlaceholderText("Filter by IP");

            // Type rapidly — input should hold the full value immediately
            await user.type(input, "192");
            expect(input).toHaveValue("192");

            // The filter icon in the IP column should not yet show the "active filter"
            // indicator until the debounce fires. mantine-datatable marks the filter
            // action icon with data-active when filtering=true.
            const ipHeader = screen
                .getAllByRole("columnheader")
                .find((h) => h.textContent?.includes("IP"))!;
            // Still within debounce window — filtering=false, no active indicator yet
            expect(
                ipHeader.querySelector('[data-active]'),
            ).toBeNull();

            // After the debounce window the active indicator appears
            await waitFor(
                () => expect(ipHeader.querySelector('[data-active]')).not.toBeNull(),
                { timeout: TEST_TIMEOUTS.MEDIUM },
            );

            void container; // suppress unused warning
        });
    });

    // ─── Outcome filter ────────────────────────────────────────────────────────

    describe("Outcome filter", () => {
        it("opens when the filter icon is clicked", async () => {
            const user = userEvent.setup();
            server.use(
                requestAuditLogHandlers.list(
                    createMockRequestAuditLogResponse({ rows: [], total: 0 }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Outcome"));

            // SegmentedControl options should be visible
            expect(await screen.findByText("Allow")).toBeInTheDocument();
            expect(screen.getByText("Deny")).toBeInTheDocument();
        });

        it("closes when filter icon is clicked again (toggle)", async () => {
            const user = userEvent.setup();
            server.use(
                requestAuditLogHandlers.list(
                    createMockRequestAuditLogResponse({ rows: [], total: 0 }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            const filterBtn = getFilterButton("Outcome");

            // Open
            await user.click(filterBtn);
            await screen.findByText("Allow");

            // Close — fireEvent.click avoids mousedown-triggered click-outside re-open
            fireEvent.click(filterBtn);
            await waitFor(() =>
                expect(screen.queryByRole("radio", { name: "Deny" })).not.toBeInTheDocument(),
            );
        });
    });

    // ─── Device filter ─────────────────────────────────────────────────────────

    describe("Device filter", () => {
        it("opens when the filter icon is clicked", async () => {
            const user = userEvent.setup();
            server.use(
                requestAuditLogHandlers.list(
                    createMockRequestAuditLogResponse({ rows: [], total: 0 }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Device"));

            expect(
                await screen.findByPlaceholderText("All devices"),
            ).toBeInTheDocument();
        });
    });

    // ─── Date range filter ─────────────────────────────────────────────────────

    describe("Date range filter", () => {
        it("opens when the filter icon is clicked and shows From and To pickers", async () => {
            const user = userEvent.setup();
            server.use(
                requestAuditLogHandlers.list(
                    createMockRequestAuditLogResponse({ rows: [], total: 0 }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Time"));

            expect(await screen.findByLabelText("From")).toBeInTheDocument();
            expect(screen.getByLabelText("To")).toBeInTheDocument();
        });
    });

    // ─── Deny reason filter ────────────────────────────────────────────────────

    describe("Deny reason filter", () => {
        it("opens when the filter icon is clicked", async () => {
            const user = userEvent.setup();
            server.use(
                requestAuditLogHandlers.list(
                    createMockRequestAuditLogResponse({ rows: [], total: 0 }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Reason"));

            expect(
                await screen.findByPlaceholderText("Any reason"),
            ).toBeInTheDocument();
        });
    });
});
