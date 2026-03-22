import { describe, expect, it } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http } from "msw";
import { server } from "@/test/setup";
import { renderWithProviders } from "@/test/utils";
import { AddressHistoryPage } from "@/pages/AddressHistoryPage";
import {
    createMockAddressHistoryResponse,
} from "@/test/mocks/data";
import { addressHandlers, endpoints, responses } from "@/test/mocks/handlers";
import { TEST_TIMEOUTS } from "@/test/constants";

const BASE_ENTRY =
    "/address-history?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z";

function renderTable(initialEntries = [BASE_ENTRY]) {
    return renderWithProviders(<AddressHistoryPage />, {
        initialEntries,
    });
}

function getFilterButton(columnTitle: string | RegExp) {
    const header = screen
        .getAllByRole("columnheader")
        .find((h) => (typeof columnTitle === "string"
            ? h.textContent?.includes(columnTitle)
            : columnTitle.test(h.textContent ?? "")));
    if (!header) throw new Error(`Column header "${columnTitle}" not found`);
    return within(header).getByRole("button");
}

describe("AddressHistoryTable", () => {
    it("renders event rows from API response", async () => {
        renderTable();

        await waitFor(
            () => expect(screen.getAllByText("10.0.0.1")).toHaveLength(2),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByText("10.0.0.2")).toBeInTheDocument();
        expect(screen.getByText("3 events")).toBeInTheDocument();
    });

    it("shows no-records message when empty", async () => {
        server.use(addressHandlers.history.empty());

        renderTable();

        await waitFor(
            () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        // Column headers remain visible for filter access
        expect(
            screen.getAllByRole("columnheader").find((h) => h.textContent?.includes("IP")),
        ).toBeDefined();
    });

    it("shows error alert on server error", async () => {
        server.use(http.get(endpoints.addressHistory, () => responses.serverError()));

        renderTable();

        await waitFor(
            () => expect(screen.getByText("Failed to load")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows status badges", async () => {
        renderTable();

        await waitFor(
            () => expect(screen.getAllByText("Enabled").length).toBeGreaterThan(0),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByText("Disabled")).toBeInTheDocument();
    });

    it("shows source badges", async () => {
        renderTable();

        await waitFor(
            () => expect(screen.getByText("Heartbeat")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByText("Manual")).toBeInTheDocument();
        expect(screen.getByText("Expiry")).toBeInTheDocument();
    });

    it("shows device name column", async () => {
        renderTable();

        await waitFor(
            () => expect(screen.getAllByText("Test Device").length).toBeGreaterThan(0),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows Load more button when next_cursor is set", async () => {
        server.use(
            addressHandlers.history.success(
                createMockAddressHistoryResponse({
                    total_events: 100,
                    next_cursor: 5,
                }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("Load more")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("hides Load more button when no next_cursor", async () => {
        server.use(
            addressHandlers.history.success(
                createMockAddressHistoryResponse({
                    total_events: 3,
                    next_cursor: null,
                }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("3 events")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.queryByText("Load more")).not.toBeInTheDocument();
    });

    // ─── IP filter ───────────────────────────────────────────────────────────

    describe("IP filter", () => {
        it("opens when the filter icon is clicked", async () => {
            const user = userEvent.setup();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("IP"));

            expect(
                await screen.findByPlaceholderText("Filter by IP"),
            ).toBeInTheDocument();
        });

        it("retains typed characters without resetting", async () => {
            const user = userEvent.setup();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("IP"));
            const input = await screen.findByPlaceholderText("Filter by IP");
            await user.type(input, "192.168.1");

            expect(input).toHaveValue("192.168.1");
        });
    });

    // ─── Status filter ───────────────────────────────────────────────────────

    describe("Status filter", () => {
        it("opens with All/Enabled/Disabled options", async () => {
            const user = userEvent.setup();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Status"));

            expect(await screen.findByText("All")).toBeInTheDocument();
        });
    });

    // ─── Source filter ───────────────────────────────────────────────────────

    describe("Source filter", () => {
        it("opens when the filter icon is clicked", async () => {
            const user = userEvent.setup();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Source"));

            expect(
                await screen.findByPlaceholderText("All sources"),
            ).toBeInTheDocument();
        });
    });

    // ─── Device filter ───────────────────────────────────────────────────────

    describe("Device filter", () => {
        it("opens when the filter icon is clicked", async () => {
            const user = userEvent.setup();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Device"));

            expect(
                await screen.findByPlaceholderText("All devices"),
            ).toBeInTheDocument();
        });
    });

    // ─── Date range filter ───────────────────────────────────────────────────

    describe("Date range filter", () => {
        it("shows From and To pickers", async () => {
            const user = userEvent.setup();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Time"));

            expect(await screen.findByLabelText("From")).toBeInTheDocument();
            expect(screen.getByLabelText("To")).toBeInTheDocument();
        });
    });
});
