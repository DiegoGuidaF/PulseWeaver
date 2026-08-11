import { describe, expect, it } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";
import { AddressHistoryPage } from "@/pages/address-history/AddressHistoryPage";
import {
    createMockAddressHistoryAtRiskDevice,
    createMockAddressHistoryBucket,
    createMockAddressHistoryEvent,
    createMockAddressHistoryHistogramResponse,
    createMockAddressHistoryResponse,
} from "@/test/mocks/data";
import { addressHandlers, endpoints, responses } from "@/test/mocks/handlers";
import { TEST_TIMEOUTS } from "@/test/constants";
import { TtlRisk } from "@/lib/api";

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

    it("queries both endpoints unfiltered on first load", async () => {
        const eventsQueries: string[] = [];
        const histogramQueries: string[] = [];
        server.use(
            http.get(endpoints.addressHistory, ({ request }) => {
                eventsQueries.push(new URL(request.url).search);
                return HttpResponse.json(createMockAddressHistoryResponse());
            }),
            http.get(endpoints.addressHistoryHistogram, ({ request }) => {
                histogramQueries.push(new URL(request.url).search);
                return HttpResponse.json(createMockAddressHistoryHistogramResponse());
            }),
        );

        renderTable(["/address-history"]);

        await waitFor(
            () => {
                expect(eventsQueries.length).toBeGreaterThan(0);
                expect(histogramQueries.length).toBeGreaterThan(0);
            },
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        // The default view narrows nothing — routine refreshes included. A seeded
        // event_kind default would hide rows the filter bar reports as unfiltered.
        expect(eventsQueries[0]).not.toContain("event_kind");
        expect(histogramQueries[0]).not.toContain("event_kind");
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

        // "Enabled"/"Disabled" render in both the Status column and the Event
        // column (an `enabled`/`disabled` event_kind always pairs with a
        // matching is_enabled), so both are expected to appear more than once.
        await waitFor(
            () => expect(screen.getAllByText("Enabled").length).toBeGreaterThan(0),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getAllByText("Disabled").length).toBeGreaterThan(0);
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

    it("shows event kind badges", async () => {
        renderTable();

        // "Created" is unambiguous — unlike Enabled/Disabled it never doubles as
        // a Status badge value, so its presence alone confirms the Event column
        // renders `event_kind` via the server enum.
        await waitFor(
            () => expect(screen.getByText("Created")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows TTL headroom badges rendered from the server enum", async () => {
        renderTable();

        await waitFor(
            () => expect(screen.getAllByText("Comfortable").length).toBeGreaterThan(0),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        // Scoped to the table: the risk chart's legend names the same bands.
        expect(within(screen.getByRole("table")).getByText("Past TTL")).toBeInTheDocument();
        expect(
            screen.getAllByRole("columnheader").find((h) => h.textContent?.includes("TTL headroom")),
        ).toBeDefined();
    });

    it("renders unknown TTL headroom as a dash, not a badge", async () => {
        server.use(
            addressHandlers.history.success(
                createMockAddressHistoryResponse({
                    events: [createMockAddressHistoryEvent({ id: 1, ttl_risk: TtlRisk.UNKNOWN })],
                    total: 1,
                }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getAllByText("—").length).toBeGreaterThan(0),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        // `unknown` is an absence of a reading, so no badge text for it appears.
        expect(screen.queryByText("Not applicable")).not.toBeInTheDocument();
        expect(screen.queryByText("Unknown")).not.toBeInTheDocument();
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
                    total: 100,
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
                    total: 3,
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

    // ─── Column management ──────────────────────────────────────────────────

    describe("Column chooser", () => {
        it("hides a column when unticked in the chooser", async () => {
            const user = setupUser();
            renderTable();

            await waitFor(
                () => expect(screen.getAllByText("10.0.0.1").length).toBeGreaterThan(0),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            expect(
                screen.getAllByRole("columnheader").find((h) => h.textContent?.includes("Source")),
            ).toBeDefined();

            await user.click(screen.getByRole("button", { name: "Columns" }));
            await user.click(await screen.findByRole("checkbox", { name: "Source" }));

            await waitFor(() =>
                expect(
                    screen.getAllByRole("columnheader").find((h) => h.textContent?.includes("Source")),
                ).toBeUndefined(),
            );
        });

        it("keeps the mandatory Time column non-removable", async () => {
            const user = setupUser();
            renderTable();

            await waitFor(
                () => expect(screen.getAllByText("10.0.0.1").length).toBeGreaterThan(0),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByRole("button", { name: "Columns" }));

            const timeCheckbox = await screen.findByRole("checkbox", { name: "Time" });
            expect(timeCheckbox).toBeChecked();
            expect(timeCheckbox).toBeDisabled();
        });
    });

    // ─── Distinct query keys (events vs histogram) ──────────────────────────

    it("does not refetch the histogram when the events cursor changes", async () => {
        let historyCalls = 0;
        let histogramCalls = 0;
        server.use(
            http.get(endpoints.addressHistory, () => {
                historyCalls++;
                return HttpResponse.json(
                    createMockAddressHistoryResponse({ total: 100, next_cursor: 5 }),
                );
            }),
            http.get(endpoints.addressHistoryHistogram, () => {
                histogramCalls++;
                return HttpResponse.json(createMockAddressHistoryHistogramResponse());
            }),
        );

        const user = setupUser();
        renderTable();

        await waitFor(
            () => expect(screen.getByRole("button", { name: "Next page" })).toBeEnabled(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        const historyCallsAfterLoad = historyCalls;
        const histogramCallsAfterLoad = histogramCalls;
        expect(histogramCallsAfterLoad).toBeGreaterThan(0);

        await user.click(screen.getByRole("button", { name: "Next page" }));

        await waitFor(
            () => expect(historyCalls).toBeGreaterThan(historyCallsAfterLoad),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        // The cursor lives outside the histogram's query key, so the page turn
        // above must not have triggered a second histogram fetch.
        expect(histogramCalls).toBe(histogramCallsAfterLoad);
    });

    // ─── IP filter ───────────────────────────────────────────────────────────

    describe("IP filter", () => {
        it("opens when the filter icon is clicked", async () => {
            const user = setupUser();
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
            const user = setupUser();
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

    // ─── Device filter ───────────────────────────────────────────────────────

    describe("Device filter", () => {
        it("opens with device and owning-user sub-filters", async () => {
            const user = setupUser();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Device"));

            expect(await screen.findByText("By device")).toBeInTheDocument();
            expect(screen.getByText("By owning user")).toBeInTheDocument();
        });
    });

    // ─── Source filter ───────────────────────────────────────────────────────

    describe("Source filter", () => {
        it("opens with a multi-select of sources", async () => {
            const user = setupUser();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Source"));

            expect(
                await screen.findByPlaceholderText("Select values"),
            ).toBeInTheDocument();
        });
    });

    // ─── Event kind filter ───────────────────────────────────────────────────

    describe("Event kind filter", () => {
        it("opens with a multi-select of event kinds", async () => {
            const user = setupUser();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Event"));

            expect(
                await screen.findByPlaceholderText("Select values"),
            ).toBeInTheDocument();
        });
    });

    // ─── TTL risk filter ─────────────────────────────────────────────────────

    describe("TTL risk filter", () => {
        it("opens with a multi-select of risk levels", async () => {
            const user = setupUser();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("TTL headroom"));

            expect(
                await screen.findByPlaceholderText("Select values"),
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
            // The device refs picker resolves independently of the address-history query.
            expect(await screen.findByText(/Test Device/)).toBeInTheDocument();
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

        it("shows a TTL headroom chip when ttl_risk filter is set", async () => {
            // No matching events means no at-risk buckets either, so the chart
            // is in its empty state and its legend cannot shadow the chip.
            server.use(
                addressHandlers.history.empty(),
                addressHandlers.historyHistogram.success(createMockAddressHistoryHistogramResponse({ buckets: [] })),
            );

            renderTable([
                "/address-history?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z&ttl_risk=breached",
            ]);

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("TTL headroom:")).toBeInTheDocument();
            expect(screen.getByText(/Past TTL/)).toBeInTheDocument();
        });

        it("shows an Event chip for a state-change selection like any other", async () => {
            server.use(addressHandlers.history.empty());

            renderTable([
                "/address-history?preset=last_24h&event_kind=created&event_kind=enabled&event_kind=disabled",
            ]);

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Event:")).toBeInTheDocument();
            expect(screen.getByText(/Created/)).toBeInTheDocument();
        });

        it("shows an Event chip once the user picks a custom event-kind subset", async () => {
            server.use(addressHandlers.history.empty());

            renderTable([
                "/address-history?preset=last_24h&event_kind=refresh",
            ]);

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Event:")).toBeInTheDocument();
            expect(screen.getByText(/Refresh/)).toBeInTheDocument();
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
            expect(screen.queryByText("Source:")).not.toBeInTheDocument();
            expect(screen.queryByText("Event:")).not.toBeInTheDocument();
            expect(screen.queryByText("TTL headroom:")).not.toBeInTheDocument();
        });

        it("removes the IP chip and clears the filter when remove is clicked", async () => {
            const user = setupUser();
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

    // ─── Tuning strip and renewal-timing chart ──────────────────────────────

    describe("Tuning strip", () => {
        it("reports an empty window as nothing having renewed, not as a clean bill of health", async () => {
            server.use(addressHandlers.historyHistogram.success(createMockAddressHistoryHistogramResponse({ buckets: [] })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No lease renewals in this window")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it("renders ranked entries and applies the device_id filter on click", async () => {
            const eventsQueries: string[] = [];
            server.use(
                http.get(endpoints.addressHistory, ({ request }) => {
                    eventsQueries.push(new URL(request.url).search);
                    return HttpResponse.json(createMockAddressHistoryResponse());
                }),
                addressHandlers.historyHistogram.success(
                    createMockAddressHistoryHistogramResponse({
                        buckets: [createMockAddressHistoryBucket({ critical_device_count: 1 })],
                        at_risk_devices: [
                            createMockAddressHistoryAtRiskDevice({
                                device_id: 7,
                                device_name: "nas-01",
                                worst_risk: TtlRisk.BREACHED,
                                renewal_count: 189,
                                late_renewal_count: 6,
                                ttl_seconds: 3600,
                                p95_gap_seconds: 4320,
                            }),
                        ],
                    }),
                ),
            );

            const user = setupUser();
            renderTable();

            await waitFor(
                () => expect(screen.getByText("nas-01")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            // Both numbers are labelled on their face — the renewal total can
            // never be misread as a tally of late renewals.
            expect(screen.getByText(/189 renewals · 6 past TTL \(3%\)/)).toBeInTheDocument();

            await user.click(screen.getByRole("button", { name: "Filter by nas-01" }));

            await waitFor(
                () => expect(eventsQueries.at(-1)).toContain("device_id=7"),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    // ─── Date range filter ───────────────────────────────────────────────────

    describe("Date range filter", () => {
        it("shows From and To pickers", async () => {
            const user = setupUser();
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
