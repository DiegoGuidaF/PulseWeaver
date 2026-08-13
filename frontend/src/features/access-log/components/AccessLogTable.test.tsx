import { beforeEach, describe, expect, it } from "vitest";
import { screen, waitFor, within, fireEvent } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";
import { AccessLogPage } from "@/pages/access-log/AccessLogPage";
import {
    createMockAccessLogRow,
    createMockAccessLogResponse,
    createMockAccessLogDetail,
} from "@/test/mocks/data";
import { endpoints, accessLogHandlers, responses } from "@/test/mocks/handlers";
import { TEST_TIMEOUTS } from "@/test/constants";

// Pre-set date range so the component starts with a bounded time window.
const BASE_ENTRY =
    "/access-log?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z";

// ─── Helpers ─────────────────────────────────────────────────────────────────

function renderTable(initialEntries = [BASE_ENTRY]) {
    return renderWithProviders(<AccessLogPage />, { initialEntries });
}

// mantine-datatable gives sortable column headers role="button", so they are not
// matched by the "columnheader" role — query the <th> elements directly.
function headerCells() {
    return Array.from(document.querySelectorAll<HTMLTableCellElement>("th"));
}

function getColumnHeader(columnTitle: string | RegExp) {
    const header = headerCells().find((h) => (typeof columnTitle === "string"
        ? h.textContent?.includes(columnTitle)
        : columnTitle.test(h.textContent ?? "")));
    if (!header) throw new Error(`Column header "${columnTitle}" not found`);
    return header;
}

/**
 * The filter trigger inside a column header — the nested button with
 * `aria-haspopup` (the header cell itself is the sort control on sortable columns).
 */
function getFilterButton(columnTitle: string | RegExp) {
    const header = getColumnHeader(columnTitle);
    const btn = within(header)
        .getAllByRole("button")
        .find((b) => b.getAttribute("aria-haspopup"));
    if (!btn) throw new Error(`Filter button for "${columnTitle}" not found`);
    return btn;
}

/** mantine-datatable keeps its loader mounted and flags it while `fetching`. */
function tableIsDimmed() {
    return document.querySelector(".mantine-datatable-loader-fetching") !== null;
}

/** The chart's LoadingOverlay mounts only while it is visible. */
function chartIsDimmed() {
    return document.querySelector(".mantine-LoadingOverlay-root") !== null;
}

/** Turn on a column that is hidden by default via the Columns chooser. */
async function showColumn(user: ReturnType<typeof setupUser>, label: string) {
    await user.click(screen.getByRole("button", { name: "Columns" }));
    await user.click(await screen.findByRole("checkbox", { name: label, hidden: true }));
    await user.keyboard("{Escape}");
}

beforeEach(() => {
    // Column-chooser visibility is persisted to localStorage; isolate each test.
    localStorage.clear();
});

// ─── Basic rendering ─────────────────────────────────────────────────────────

describe("AccessLogTable", () => {
    it("renders rows with contributor user and device", async () => {
        const user = setupUser();
        const row = createMockAccessLogRow({
            client_ip: "203.0.113.42",
            target_host: "example.com",
            outcome: true,
        });
        server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

        renderTable();

        await waitFor(
            () => expect(screen.getByText("203.0.113.42")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.getByText("example.com")).toBeInTheDocument();
        expect(screen.getByText("Test User")).toBeInTheDocument();
        // "Authorized by" is hidden by default; enable it to assert the device.
        await showColumn(user, "Authorized by");
        expect(screen.getByText("Test Device")).toBeInTheDocument();
        expect(screen.getAllByText("Allow").length).toBeGreaterThan(0);
        expect(screen.getByText("1 result")).toBeInTheDocument();
    });

    it("renders an em dash in the User column when there are no contributors", async () => {
        const row = createMockAccessLogRow({ client_ip: "10.9.9.9", contributors: [], contributor_count: 0 });
        server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

        renderTable();

        await waitFor(
            () => expect(screen.getByText("10.9.9.9")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
        expect(screen.queryByText("Test User")).not.toBeInTheDocument();
    });

    it("shows no-records message while keeping column headers visible", async () => {
        server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

        renderTable();

        await waitFor(
            () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(getColumnHeader("Time")).toBeDefined();
        expect(getColumnHeader("IP")).toBeDefined();
        expect(getColumnHeader("Decision")).toBeDefined();
    });

    it("shows error alert when API returns 500", async () => {
        server.use(http.get(endpoints.accessLog, () => responses.serverError()));

        renderTable();

        await waitFor(
            () => expect(screen.getByText("Failed to load")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows error alert when API returns 403", async () => {
        server.use(
            http.get(endpoints.accessLog, () =>
                responses.forbidden({ message: "Forbidden - admin credentials required" }),
            ),
        );

        renderTable();

        await waitFor(
            () => expect(screen.getByText("Failed to load")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("row click opens detail drawer with contributors", async () => {
        const user = setupUser();
        const row = createMockAccessLogRow({
            id: 42,
            client_ip: "10.0.0.1",
            outcome: false,
            deny_reason: "invalid_token",
            target_host: "secure.example.com",
        });
        server.use(
            accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })),
            accessLogHandlers.entry.success(createMockAccessLogDetail({ ...row })),
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
        expect(screen.getByText("Contributors")).toBeInTheDocument();
        expect(screen.getAllByText("Invalid token").length).toBeGreaterThan(0);
        expect(screen.getAllByText("secure.example.com").length).toBeGreaterThan(0);
    });

    // ─── Column filter popovers ───────────────────────────────────────────────

    describe("IP filter", () => {
        it("opens with an operator selector and a value input", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("IP"));

            expect(await screen.findByDisplayValue("is any of")).toBeInTheDocument();
            expect(screen.getByPlaceholderText("Type and press Enter")).toBeInTheDocument();
        });

        it("keeps a non-default operator selected even before any value is entered", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("IP"));

            const operator = await screen.findByDisplayValue("is any of");
            await user.click(operator);
            await user.click(await screen.findByText("is none of"));

            // Without a persisted operator the selector would snap back to "is any of".
            await waitFor(() =>
                expect(screen.getByDisplayValue("is none of")).toBeInTheDocument(),
            );
        });

        it("closes when the filter icon is clicked again (toggle)", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            const filterBtn = getFilterButton("IP");

            await user.click(filterBtn);
            expect(await screen.findByPlaceholderText("Type and press Enter")).toBeInTheDocument();

            // fireEvent.click avoids the mousedown click-outside that would reopen the popover
            fireEvent.click(filterBtn);
            await waitFor(() =>
                expect(screen.queryByPlaceholderText("Type and press Enter")).not.toBeInTheDocument(),
            );
        });
    });

    describe("Decision filter", () => {
        it("opens with Allow/Deny options and a deny-reason section", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Decision"));

            expect(await screen.findByText("Allow")).toBeInTheDocument();
            expect(screen.getByText("Deny")).toBeInTheDocument();
            expect(screen.getByText("Deny reason")).toBeInTheDocument();
        });
    });

    describe("Authorized-by filter", () => {
        it("opens with device and network-policy sections", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await showColumn(user, "Authorized by");
            await user.click(getFilterButton("Authorized by"));

            expect(await screen.findByText("By device")).toBeInTheDocument();
            expect(screen.getByText("By network policy")).toBeInTheDocument();
        });

        // The popover hosts two ColumnFilters that each commit on close. Both
        // must survive: react-router branches every functional search-param
        // update off the same render-time snapshot, so without composing them
        // the second commit (network policy) would clobber the first (device).
        it("applies both device and network-policy filters set in one session", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await showColumn(user, "Authorized by");
            await user.click(getFilterButton("Authorized by"));

            const deviceSection = (await screen.findByText("By device")).parentElement!;
            await user.click(within(deviceSection).getByPlaceholderText("Select values"));
            await user.click(await screen.findByRole("option", { name: "Test Device" }));

            const policySection = screen.getByText("By network policy").parentElement!;
            await user.click(within(policySection).getByPlaceholderText("Select values"));
            await user.click(await screen.findByRole("option", { name: /Test Policy/ }));

            // Apply commits both ColumnFilters as the popover unmounts.
            await user.click(screen.getByRole("button", { name: "Apply" }));

            expect(await screen.findByText("Device:")).toBeInTheDocument();
            expect(screen.getByText(/Test Device/)).toBeInTheDocument();
            expect(screen.getByText("Network policy:")).toBeInTheDocument();
            expect(screen.getByText(/Test Policy/)).toBeInTheDocument();
        });
    });

    describe("Date range filter", () => {
        it("opens and shows From and To pickers", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

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

    // ─── Active filter chips ──────────────────────────────────────────────────

    describe("Active filter chips", () => {
        it("shows a Time chip when from/to are set", async () => {
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Time:")).toBeInTheDocument();
        });

        it("shows an IP chip with operator phrasing from a URL param", async () => {
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable([`${BASE_ENTRY}&client_ip=10.0.0.5`]);

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("IP:")).toBeInTheDocument();
            expect(screen.getByText(/is any of 10\.0\.0\.5/)).toBeInTheDocument();
        });

        it("shows an 'is unknown' Country chip for the is_null operator", async () => {
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable([`${BASE_ENTRY}&country_code_op=is_null`]);

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Country:")).toBeInTheDocument();
            expect(screen.getByText("is unknown")).toBeInTheDocument();
        });

        it("resolves a Device chip to the device name", async () => {
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable([`${BASE_ENTRY}&device_id=1`]);

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Device:")).toBeInTheDocument();
            // The device refs picker resolves independently of the access-log query.
            expect(await screen.findByText(/Test Device/)).toBeInTheDocument();
        });

        it("shows an Outcome chip when outcome filter is set", async () => {
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable([`${BASE_ENTRY}&outcome=deny`]);

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByText("Outcome:")).toBeInTheDocument();
            expect(screen.getByText(/Deny/)).toBeInTheDocument();
        });

        it("removes the IP chip and clears the filter when remove is clicked", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable([`${BASE_ENTRY}&client_ip=10.0.0.5`]);

            await waitFor(
                () => expect(screen.getByText("IP:")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            const ipPill = screen.getByText("IP:").closest(".mantine-Pill-root");
            const removeBtn = ipPill?.querySelector(".mantine-Pill-remove") as HTMLElement;
            expect(removeBtn).toBeTruthy();
            await user.click(removeBtn);

            await waitFor(() => expect(screen.queryByText("IP:")).not.toBeInTheDocument());
        });

        it("does not render chips when no filters are active", async () => {
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable(["/access-log?preset=last_24h"]);

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.queryByText("Time:")).not.toBeInTheDocument();
            expect(screen.queryByText("IP:")).not.toBeInTheDocument();
            expect(screen.queryByText("Outcome:")).not.toBeInTheDocument();
        });
    });

    // ─── Country column ───────────────────────────────────────────────────────

    describe("Country column", () => {
        it("renders flag emoji and country code when present", async () => {
            const row = createMockAccessLogRow({ client_ip: "8.8.8.8", country_code: "DE" });
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText(/🇩🇪/)).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            expect(screen.getByText(/DE/)).toBeInTheDocument();
        });

        it("renders no country code text when absent", async () => {
            const row = createMockAccessLogRow({ client_ip: "192.168.1.1", country_code: undefined });
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("192.168.1.1")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            expect(screen.queryByText("DE")).not.toBeInTheDocument();
        });
    });

    // ─── Inline filter affordance ─────────────────────────────────────────────

    describe("Inline filter affordance", () => {
        it("applies a column filter from a cell's hover filter control", async () => {
            const user = setupUser();
            const row = createMockAccessLogRow({ client_ip: "8.8.4.4", country_code: "DE" });
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("8.8.4.4")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // The control is opacity-hidden until row hover, but stays in the DOM.
            await user.click(screen.getByRole("button", { name: "Filter by this country" }));

            expect(await screen.findByText("Country:")).toBeInTheDocument();
        });
    });

    // ─── Detail drawer ────────────────────────────────────────────────────────

    describe("Detail drawer", () => {
        it("shows the Location section with ASN from the detail endpoint", async () => {
            const user = setupUser();
            const row = createMockAccessLogRow({ id: 7, client_ip: "8.8.8.8", country_code: "US" });
            server.use(
                accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })),
                // The geo detail lives only on the by-id endpoint; the list row
                // carries the country code alone.
                accessLogHandlers.entry.success(
                    createMockAccessLogDetail({
                        ...row,
                        country_name: "United States",
                        continent_code: "NA",
                        asn: 15169,
                        asn_org: "Google LLC",
                    }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("8.8.8.8")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByRole("button", { name: "View details" }));

            await waitFor(
                () => expect(screen.getByText("Location")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            expect(screen.getByText(/Google LLC/)).toBeInTheDocument();
            expect(screen.getByText(/15169/)).toBeInTheDocument();
        });

        it("renders headers fetched with the detail, which the list row never carries", async () => {
            const user = setupUser();
            const row = createMockAccessLogRow({ id: 7, client_ip: "8.8.8.8" });
            server.use(
                accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })),
                accessLogHandlers.entry.success(
                    createMockAccessLogDetail({ ...row, headers: { "User-Agent": ["pulseweaver-test"] } }),
                ),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("8.8.8.8")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByRole("button", { name: "View details" }));

            expect(await screen.findByText(/pulseweaver-test/)).toBeInTheDocument();
        });

        // The drawer fetches by id rather than reading the current page, so a link
        // to a request that is not on the loaded page still renders full detail.
        it("resolves a ?request= deep link to a row outside the current page", async () => {
            server.use(
                accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })),
                accessLogHandlers.entry.success(
                    createMockAccessLogDetail({ id: 999, client_ip: "198.51.100.7" }),
                ),
            );

            renderTable([`${BASE_ENTRY}&request=999`]);

            expect(await screen.findByText("Request Detail")).toBeInTheDocument();
            expect(await screen.findByText("198.51.100.7")).toBeInTheDocument();
        });

        it("shows an unavailable state when the entry has been pruned", async () => {
            server.use(
                accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })),
                accessLogHandlers.entry.notFound(),
            );

            renderTable([`${BASE_ENTRY}&request=999`]);

            expect(
                await screen.findByText("This request is no longer available"),
            ).toBeInTheDocument();
        });

        // Only a 404 means "gone". Any other failure is worth retrying, so it
        // must reach ErrorState rather than the terminal unavailable copy.
        it("shows a retryable error when the detail fetch fails outright", async () => {
            server.use(
                accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })),
                http.get(endpoints.accessLogEntry, () => responses.serverError()),
            );

            renderTable([`${BASE_ENTRY}&request=999`]);

            expect(await screen.findByText("Failed to load")).toBeInTheDocument();
            expect(
                screen.queryByText("This request is no longer available"),
            ).not.toBeInTheDocument();
        });

        // `Number("abc")` is NaN, which never equals itself — left unguarded it
        // re-triggers the drawer's render-phase state sync until React bails out
        // and the whole page falls into the router error boundary.
        it("ignores a ?request= param that is not a request id", async () => {
            const row = createMockAccessLogRow({ id: 1, client_ip: "10.0.0.1" });
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

            renderTable([`${BASE_ENTRY}&request=abc`]);

            await waitFor(
                () => expect(screen.getByText("10.0.0.1")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            expect(screen.queryByText("Request Detail")).not.toBeInTheDocument();
        });
    });

    // ─── Chart / table agreement ──────────────────────────────────────────────

    describe("Histogram", () => {
        // The chart's sums only reconcile with the table's total if both scans see
        // the same predicate over the same window.
        it("sends the table's filters and window to the histogram", async () => {
            const listUrls: string[] = [];
            const histogramUrls: string[] = [];
            server.use(
                http.get(endpoints.accessLog, ({ request }) => {
                    listUrls.push(request.url);
                    return HttpResponse.json(createMockAccessLogResponse({ rows: [], total: 0 }));
                }),
                http.get(endpoints.accessLogHistogram, ({ request }) => {
                    histogramUrls.push(request.url);
                    return HttpResponse.json({ buckets: [] });
                }),
            );

            renderTable([`${BASE_ENTRY}&client_ip=10.0.0.5&outcome=deny`]);

            await waitFor(
                () => expect(histogramUrls.length).toBeGreaterThan(0),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            const histogram = new URL(histogramUrls.at(-1)!).searchParams;
            const listWindows = listUrls.map((u) => new URL(u).searchParams.get("from"));

            expect(histogram.getAll("client_ip")).toEqual(["10.0.0.5"]);
            expect(histogram.get("outcome")).toBe("false");
            // A preset window is resolved against the clock, so compare against
            // every window the list asked for rather than only its latest.
            expect(listWindows).toContain(histogram.get("from"));
        });

        // Sort belongs to the paged list; the histogram has no ordering, so a
        // sort change must not re-scan it.
        it("never sends sort, order or cursor to the histogram", async () => {
            const requestedUrls: string[] = [];
            server.use(
                accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })),
                http.get(endpoints.accessLogHistogram, ({ request }) => {
                    requestedUrls.push(request.url);
                    return HttpResponse.json({ buckets: [] });
                }),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            fireEvent.click(within(getColumnHeader("IP")).getByText("IP"));

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(requestedUrls.every((u) => !u.includes("sort=") && !u.includes("cursor="))).toBe(true);
        });

        // Keeping sort out of the histogram's params is not enough on its own: a
        // preset window re-resolved against the clock would move `from` on the
        // re-render the sort triggers, minting a new key and re-scanning anyway.
        it("does not re-scan the histogram when the sort changes", async () => {
            const histogramRequests: string[] = [];
            server.use(
                accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })),
                http.get(endpoints.accessLogHistogram, ({ request }) => {
                    histogramRequests.push(request.url);
                    return HttpResponse.json({ buckets: [] });
                }),
            );

            renderTable(["/access-log?preset=last_24h"]);

            await waitFor(
                () => expect(histogramRequests).toHaveLength(1),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            fireEvent.click(within(getColumnHeader("IP")).getByText("IP"));

            await waitFor(
                () => expect(within(getColumnHeader("IP")).getByLabelText(/^Sorted/)).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.MEDIUM },
            );
            expect(histogramRequests).toHaveLength(1);
        });

        it("surfaces a histogram failure without hiding the table", async () => {
            const row = createMockAccessLogRow({ client_ip: "10.0.0.1" });
            server.use(
                accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })),
                http.get(endpoints.accessLogHistogram, () => responses.serverError()),
            );

            renderTable();

            expect(await screen.findByText("Failed to load traffic")).toBeInTheDocument();
            expect(screen.getByText("10.0.0.1")).toBeInTheDocument();
            expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
        });
    });

    // ─── Refresh affordances ──────────────────────────────────────────────────

    describe("Refresh affordances", () => {
        // Long enough that an assertion lands inside the in-flight window.
        const SLOW_MS = 400;

        /**
         * Answers the first request per endpoint at once and stalls every later
         * one, so a refetch has an observable in-flight window. Uses a preset
         * window, whose resolved `from` is pinned to the params — an inline
         * clock read would mint a new query key on every render and dim both
         * panels for reasons unrelated to what the test is driving.
         */
        function renderWithSlowRefetches() {
            let listCount = 0;
            let histogramCount = 0;
            const row = createMockAccessLogRow({ client_ip: "10.0.0.1" });
            server.use(
                http.get(endpoints.accessLog, async () => {
                    if (listCount++ > 0) await delay(SLOW_MS);
                    return HttpResponse.json(createMockAccessLogResponse({ rows: [row], total: 1 }));
                }),
                http.get(endpoints.accessLogHistogram, async () => {
                    if (histogramCount++ > 0) await delay(SLOW_MS);
                    return HttpResponse.json({ buckets: [] });
                }),
            );
            renderTable(["/access-log?preset=last_24h"]);
            return {
                settled: () =>
                    waitFor(() => expect(screen.getByText("10.0.0.1")).toBeInTheDocument(), {
                        timeout: TEST_TIMEOUTS.SHORT,
                    }),
                listRequests: () => listCount,
            };
        }

        it("dims both panels while a refresh the user clicked is in flight", async () => {
            const user = setupUser();
            const scenario = renderWithSlowRefetches();
            await scenario.settled();
            expect(tableIsDimmed()).toBe(false);

            await user.click(screen.getByRole("button", { name: "Refresh" }));

            await waitFor(() => expect(chartIsDimmed()).toBe(true));
            expect(tableIsDimmed()).toBe(true);

            await waitFor(() => expect(chartIsDimmed()).toBe(false), {
                timeout: TEST_TIMEOUTS.MEDIUM,
            });
            expect(tableIsDimmed()).toBe(false);
        });

        it("dims both panels while a new time window is in flight", async () => {
            const user = setupUser();
            const scenario = renderWithSlowRefetches();
            await scenario.settled();

            await user.selectOptions(screen.getByLabelText("Time range"), "last_1w");

            await waitFor(() => expect(chartIsDimmed()).toBe(true));
            expect(tableIsDimmed()).toBe(true);
        });

        // A poll nobody asked for must repaint in place: dimming the page every
        // interval is the flash the loading-state convention exists to prevent.
        it("dims neither panel during an auto-refresh poll", async () => {
            const user = setupUser();
            const scenario = renderWithSlowRefetches();
            await scenario.settled();

            await user.selectOptions(screen.getByLabelText("Auto-refresh interval"), "1000");

            // The stalled poll is still open, so this asserts mid-flight.
            await waitFor(() => expect(scenario.listRequests()).toBeGreaterThan(1), {
                timeout: TEST_TIMEOUTS.MEDIUM,
            });
            expect(tableIsDimmed()).toBe(false);
            expect(chartIsDimmed()).toBe(false);
        });

        // Sort belongs to the paged list alone, so the chart has nothing in
        // flight and must not dim alongside the table.
        it("dims only the table when the sort changes", async () => {
            const scenario = renderWithSlowRefetches();
            await scenario.settled();

            fireEvent.click(within(getColumnHeader("IP")).getByText("IP"));

            await waitFor(() => expect(tableIsDimmed()).toBe(true));
            expect(chartIsDimmed()).toBe(false);
        });
    });

    // ─── DB-IP attribution ────────────────────────────────────────────────────

    describe("DB-IP attribution", () => {
        it("renders the attribution link when a row has country_code", async () => {
            const row = createMockAccessLogRow({ client_ip: "8.8.8.8", country_code: "US" });
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("IP Geolocation by DB-IP")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            expect(screen.getByRole("link", { name: "IP Geolocation by DB-IP" })).toHaveAttribute(
                "href",
                "https://db-ip.com",
            );
        });

        it("does not render attribution when no rows have country_code", async () => {
            const row = createMockAccessLogRow({ client_ip: "192.168.1.1", country_code: undefined });
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("192.168.1.1")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            expect(screen.queryByText("IP Geolocation by DB-IP")).not.toBeInTheDocument();
        });
    });

    // ─── Column chooser ───────────────────────────────────────────────────────

    describe("Column chooser", () => {
        it("reveals the Method column when toggled on", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // Method column is opt-in — hidden by default
            expect(headerCells().some((h) => h.textContent?.includes("Method"))).toBe(false);

            await user.click(screen.getByRole("button", { name: "Columns" }));
            // The chooser lives in a Menu dropdown that jsdom treats as
            // a11y-hidden, so toggle via the checkbox's visible label text.
            await user.click(await screen.findByText("Method"));

            await waitFor(() =>
                expect(headerCells().some((h) => h.textContent?.includes("Method"))).toBe(true),
            );
        });
    });

    // ─── Sorting ──────────────────────────────────────────────────────────────

    describe("Sorting", () => {
        it("requests the chosen sort column and direction", async () => {
            const requestedUrls: string[] = [];
            server.use(
                http.get(endpoints.accessLog, ({ request }) => {
                    requestedUrls.push(request.url);
                    return HttpResponse.json(createMockAccessLogResponse({ rows: [], total: 0 }));
                }),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // The sortable header cell is itself the sort control.
            fireEvent.click(within(getColumnHeader("IP")).getByText("IP"));

            await waitFor(
                () => expect(requestedUrls.some((u) => u.includes("sort=client_ip"))).toBe(true),
                { timeout: TEST_TIMEOUTS.MEDIUM },
            );
        });

        it("clears sorting after the third click on a column (asc → desc → off)", async () => {
            const requestedUrls: string[] = [];
            server.use(
                http.get(endpoints.accessLog, ({ request }) => {
                    requestedUrls.push(request.url);
                    return HttpResponse.json(createMockAccessLogResponse({ rows: [], total: 0 }));
                }),
            );

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            const ipHeader = within(getColumnHeader("IP")).getByText("IP");

            fireEvent.click(ipHeader);
            await waitFor(
                () => expect(requestedUrls.some((u) => u.includes("sort=client_ip") && u.includes("order=asc"))).toBe(true),
                { timeout: TEST_TIMEOUTS.MEDIUM },
            );

            fireEvent.click(ipHeader);
            await waitFor(
                () => expect(requestedUrls.some((u) => u.includes("sort=client_ip") && u.includes("order=desc"))).toBe(true),
                { timeout: TEST_TIMEOUTS.MEDIUM },
            );

            // Third click cycles off, back to the default newest-first sort. That
            // query was already fetched on mount, so it is served from cache and
            // issues no request — assert the rendered sort state, not the network.
            fireEvent.click(ipHeader);
            await waitFor(
                () =>
                    expect(
                        within(getColumnHeader("IP")).getByLabelText("Not sorted"),
                    ).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.MEDIUM },
            );
            expect(within(getColumnHeader("Time")).getByLabelText("Sorted descending")).toBeInTheDocument();
        });
    });

    // ─── Column chooser (hiding) ──────────────────────────────────────────────

    describe("Column chooser hiding", () => {
        it("hides a default-visible column when toggled off", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(headerCells().some((h) => h.textContent?.includes("Country"))).toBe(true);

            await user.click(screen.getByRole("button", { name: "Columns" }));
            // "Country" is also the visible column header, so target the checkbox
            // by role (the Menu dropdown is a11y-hidden in jsdom).
            await user.click(await screen.findByRole("checkbox", { name: "Country", hidden: true }));

            await waitFor(() =>
                expect(headerCells().some((h) => h.textContent?.includes("Country"))).toBe(false),
            );
        });

        it("keeps Time mandatory but lets other columns be toggled", async () => {
            const user = setupUser();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByRole("button", { name: "Columns" }));

            const timeCheckbox = await screen.findByRole("checkbox", { name: "Time", hidden: true });
            expect(timeCheckbox).toBeDisabled();
            expect(timeCheckbox).toBeChecked();

            const hostCheckbox = await screen.findByRole("checkbox", { name: "Host", hidden: true });
            expect(hostCheckbox).toBeEnabled();
            expect(hostCheckbox).toBeChecked();
        });
    });
});
