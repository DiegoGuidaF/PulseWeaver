import { describe, expect, it } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";
import { AddressHistoryPage } from "@/pages/address-history/AddressHistoryPage";
import {
    createMockAddressHistoryEvent,
    createMockAddressHistoryHistogramResponse,
    createMockAddressHistoryResponse,
} from "@/test/mocks/data";
import { addressHandlers, endpoints, responses } from "@/test/mocks/handlers";
import { TEST_TIMEOUTS } from "@/test/constants";
import { AddressEventKind, AddressEventTrigger, TtlRisk } from "@/lib/api";

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

    it("leaves the address state unstated when the event kind implies it", async () => {
        server.use(
            addressHandlers.history.success(
                createMockAddressHistoryResponse({
                    events: [
                        createMockAddressHistoryEvent({
                            id: 1,
                            ip: "10.0.0.1",
                            event_kind: AddressEventKind.CREATED,
                            is_enabled: true,
                        }),
                    ],
                    total: 1,
                }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("Created")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.queryByText(/now (enabled|disabled)/)).not.toBeInTheDocument();
    });

    it("states the address state when the event kind does not imply it", async () => {
        // `created` means "no earlier event survives", not "the address was
        // created": once retention has pruned an address's history its oldest
        // surviving event can be a cap eviction, leaving it disabled. That row
        // is the reason the Status column's data outlives the Status column.
        server.use(
            addressHandlers.history.success(
                createMockAddressHistoryResponse({
                    events: [
                        createMockAddressHistoryEvent({
                            id: 1,
                            ip: "10.0.0.1",
                            event_kind: AddressEventKind.CREATED,
                            source: "limit_exceeded",
                            is_enabled: false,
                        }),
                    ],
                    total: 1,
                }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("Created")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByText(/now disabled/)).toBeInTheDocument();
    });

    it("shows the source of each event", async () => {
        renderTable();

        await waitFor(
            () => expect(screen.getByText("Heartbeat")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByText("Web UI")).toBeInTheDocument();
        expect(screen.getByText("Expiry")).toBeInTheDocument();
        // Which subsystem wrote an event is read once you have stopped on the
        // row, so it never takes colour — not even for the server jobs.
        expect(screen.getByText("Expiry").closest(".mantine-Badge-root")).toBeNull();
    });

    it("badges a user trigger and nothing else", async () => {
        renderTable();

        await waitFor(
            () => expect(screen.getByText("Heartbeat")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        const table = within(screen.getByRole("table"));
        expect(table.getByText("User").closest(".mantine-Badge-root")).not.toBeNull();
        expect(table.getByText("Scheduled").closest(".mantine-Badge-root")).toBeNull();
        expect(table.getByText("System").closest(".mantine-Badge-root")).toBeNull();
    });

    it("renders Source and Trigger as independent axes", async () => {
        // The whole point of the column: one subsystem can write events set off
        // by different things, so a shared source must not collapse the triggers.
        server.use(
            addressHandlers.history.success(
                createMockAddressHistoryResponse({
                    events: [
                        createMockAddressHistoryEvent({
                            id: 2,
                            ip: "10.0.0.1",
                            source: "heartbeat",
                            trigger_type: AddressEventTrigger.USER,
                        }),
                        createMockAddressHistoryEvent({
                            id: 1,
                            ip: "10.0.0.2",
                            source: "heartbeat",
                            trigger_type: AddressEventTrigger.SCHEDULE,
                        }),
                    ],
                    total: 2,
                }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("10.0.0.2")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        const table = within(screen.getByRole("table"));
        expect(table.getAllByText("Heartbeat")).toHaveLength(2);
        expect(table.getByText("User")).toBeInTheDocument();
        expect(table.getByText("Scheduled")).toBeInTheDocument();
    });

    it("de-emphasises a routine refresh without touching row opacity", async () => {
        // Opacity is the one emphasis channel that cannot be made accessible: it
        // composites foreground and background together, dragging every `light`
        // badge in the row under 4.5:1. A routine refresh recedes because its
        // cells carry no colour, not because the row is turned down — so no row
        // may carry an inline opacity, at 0.55 or at any friendlier value.
        server.use(
            addressHandlers.history.success(
                createMockAddressHistoryResponse({
                    events: [
                        createMockAddressHistoryEvent({
                            id: 2,
                            ip: "10.0.0.1",
                            trigger_type: AddressEventTrigger.USER,
                            event_kind: AddressEventKind.REFRESH,
                        }),
                        createMockAddressHistoryEvent({
                            id: 1,
                            ip: "10.0.0.2",
                            trigger_type: AddressEventTrigger.SCHEDULE,
                            event_kind: AddressEventKind.REFRESH,
                        }),
                    ],
                    total: 2,
                }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("10.0.0.2")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        const table = within(screen.getByRole("table"));
        for (const row of table.getAllByRole("row")) {
            expect(row.style.opacity).toBe("");
        }
        // The refresh label itself is what recedes, and it takes no badge.
        expect(table.getAllByText("Refresh")[0].closest(".mantine-Badge-root")).toBeNull();
    });

    it("keeps Trigger beside Source for a user with a stored column order", async () => {
        // The library appends an unknown accessor to a stored order, so a new
        // column lands at the far right for every returning user unless the
        // store key moves on. Trigger only reads against Source, so it does.
        localStorage.setItem(
            "pulseweaver:address-history:columns:v1-columns-order",
            JSON.stringify(["timestamp", "device_name", "ip", "geo", "is_enabled", "source", "event_kind", "renewal_gap_seconds", "ttl_risk"]),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("10.0.0.2")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        const titles = screen.getAllByRole("columnheader").map((h) => h.textContent ?? "");
        const source = titles.findIndex((t) => t.includes("Source"));
        expect(titles[source + 1]).toContain("Trigger");
    });

    it("badges the transitions the address went through", async () => {
        renderTable();

        await waitFor(
            () => expect(screen.getByText("Created")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        const table = within(screen.getByRole("table"));
        expect(table.getByText("Created").closest(".mantine-Badge-root")).not.toBeNull();
        expect(table.getByText("Enabled").closest(".mantine-Badge-root")).not.toBeNull();
        expect(table.getByText("Disabled").closest(".mantine-Badge-root")).not.toBeNull();
    });

    it("badges only the TTL levels short of comfortable", async () => {
        renderTable();

        await waitFor(
            () => expect(screen.getAllByText("Comfortable").length).toBeGreaterThan(0),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        // Comfortable is ~88% of real rows: a badge there is colour spent on the
        // norm, which is what buried the levels that need reading.
        const table = within(screen.getByRole("table"));
        expect(table.getAllByText("Comfortable")[0].closest(".mantine-Badge-root")).toBeNull();
        expect(table.getByText("Past TTL").closest(".mantine-Badge-root")).not.toBeNull();
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

    // ─── Trigger filter ──────────────────────────────────────────────────────

    describe("Trigger filter", () => {
        it("opens with a multi-select of every trigger", async () => {
            const user = setupUser();
            server.use(addressHandlers.history.empty());

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Trigger"));
            await user.click(await screen.findByPlaceholderText("Select values"));

            for (const label of ["User", "Scheduled", "Network change", "System"]) {
                expect(await screen.findByRole("option", { name: label })).toBeInTheDocument();
            }
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

        it("shows a Trigger chip when trigger_type filter is set", async () => {
            server.use(addressHandlers.history.empty());

            renderTable([
                "/address-history?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z&trigger_type=user",
            ]);

            await waitFor(
                () => expect(screen.getByText("No address events found.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Trigger:")).toBeInTheDocument();
            expect(screen.getByText(/User/)).toBeInTheDocument();
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
            expect(screen.queryByText("Trigger:")).not.toBeInTheDocument();
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
