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
        expect(screen.getByText("3 results")).toBeInTheDocument();
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

    it("shows pagination with next page when next_cursor is set", async () => {
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
            () => expect(screen.getByRole("button", { name: "Next page" })).toBeEnabled(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("disables next page when no next_cursor", async () => {
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
            () => expect(screen.getByText("3 results")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
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

    // ─── Active filter chips ─────────────────────────────────────────────────

    describe("Active filter chips", () => {
        it("shows a Time chip when from/to are set", async () => {
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Time:")).toBeInTheDocument();
        });

        it("shows an IP chip when ip filter is set via URL", async () => {
            server.use(addressHandlers.history.empty());

            renderTable([
                "/address-history?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z&ip=10.0.0",
            ]);

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("IP:")).toBeInTheDocument();
            expect(screen.getByText(/10\.0\.0/)).toBeInTheDocument();
        });

        it("shows a Device chip with device name when device_id is set", async () => {
            server.use(addressHandlers.history.empty());

            renderTable([
                "/address-history?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z&device_id=1",
            ]);

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Device:")).toBeInTheDocument();
            expect(screen.getByText(/Test Device/)).toBeInTheDocument();
        });

        it("shows a Status chip when is_enabled filter is set", async () => {
            server.use(addressHandlers.history.empty());

            renderTable([
                "/address-history?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z&is_enabled=true",
            ]);

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Status:")).toBeInTheDocument();
        });

        it("shows a Source chip when source filter is set", async () => {
            server.use(addressHandlers.history.empty());

            renderTable([
                "/address-history?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z&source=heartbeat",
            ]);

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Source:")).toBeInTheDocument();
            expect(screen.getByText(/Heartbeat/)).toBeInTheDocument();
        });

        it("does not render chips when only preset is active", async () => {
            server.use(addressHandlers.history.empty());

            renderTable(["/address-history?preset=last_24h"]);

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.queryByText("Time:")).not.toBeInTheDocument();
            expect(screen.queryByText("IP:")).not.toBeInTheDocument();
            expect(screen.queryByText("Device:")).not.toBeInTheDocument();
            expect(screen.queryByText("Status:")).not.toBeInTheDocument();
            expect(screen.queryByText("Source:")).not.toBeInTheDocument();
        });

        it("removes the IP chip and clears the filter when remove is clicked", async () => {
            const user = userEvent.setup();
            server.use(addressHandlers.history.empty());

            renderTable([
                "/address-history?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z&ip=10.0.0",
            ]);

            await waitFor(
                () => expect(screen.getByText("IP:")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            const ipPill = screen.getByText("IP:").closest(".mantine-Pill-root");
            const removeBtn = ipPill?.querySelector(".mantine-Pill-remove") as HTMLElement;
            expect(removeBtn).toBeTruthy();
            await user.click(removeBtn);

            await waitFor(() =>
                expect(screen.queryByText("IP:")).not.toBeInTheDocument(),
            );
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
