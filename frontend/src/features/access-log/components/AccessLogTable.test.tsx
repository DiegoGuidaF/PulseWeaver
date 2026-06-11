import { beforeEach, describe, expect, it } from "vitest";
import { screen, waitFor, within, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/setup";
import { renderWithProviders } from "@/test/utils";
import { AccessLogPage } from "@/pages/access-log/AccessLogPage";
import {
    createMockAccessLogRow,
    createMockAccessLogResponse,
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

beforeEach(() => {
    // Column-chooser visibility is persisted to localStorage; isolate each test.
    localStorage.clear();
});

// ─── Basic rendering ─────────────────────────────────────────────────────────

describe("AccessLogTable", () => {
    it("renders rows with contributor user and device", async () => {
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
        expect(getColumnHeader("Outcome")).toBeDefined();
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
        const user = userEvent.setup();
        const row = createMockAccessLogRow({
            id: 42,
            client_ip: "10.0.0.1",
            outcome: false,
            deny_reason: "invalid_token",
            target_host: "secure.example.com",
        });
        server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

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
            const user = userEvent.setup();
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
            const user = userEvent.setup();
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
            const user = userEvent.setup();
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

    describe("Outcome filter", () => {
        it("opens with Allow/Deny options", async () => {
            const user = userEvent.setup();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Outcome"));

            expect(await screen.findByText("Allow")).toBeInTheDocument();
            expect(screen.getByText("Deny")).toBeInTheDocument();
        });
    });

    describe("Authorized-by filter", () => {
        it("opens with device and network-policy sections", async () => {
            const user = userEvent.setup();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(getFilterButton("Authorized by"));

            expect(await screen.findByText("By device")).toBeInTheDocument();
            expect(screen.getByText("By network policy")).toBeInTheDocument();
        });
    });

    describe("Date range filter", () => {
        it("opens and shows From and To pickers", async () => {
            const user = userEvent.setup();
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
            expect(screen.getByText(/Test Device/)).toBeInTheDocument();
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
            const user = userEvent.setup();
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
            const row = createMockAccessLogRow({ client_ip: "8.8.8.8", country_code: "DE", country_name: "Germany" });
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

    // ─── Detail drawer — Location section ─────────────────────────────────────

    describe("Detail drawer — Location section", () => {
        it("shows the Location section with ASN when GeoIP data is present", async () => {
            const user = userEvent.setup();
            const row = createMockAccessLogRow({
                client_ip: "8.8.8.8",
                country_code: "US",
                country_name: "United States",
                continent_code: "NA",
                asn: 15169,
                asn_org: "Google LLC",
            });
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [row], total: 1 })));

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
            const user = userEvent.setup();
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

            // Third click cycles off — the request falls back to the default sort.
            fireEvent.click(ipHeader);
            await waitFor(
                () => expect(requestedUrls.at(-1)?.includes("sort=client_ip")).toBe(false),
                { timeout: TEST_TIMEOUTS.MEDIUM },
            );
        });
    });

    // ─── Column chooser (hiding) ──────────────────────────────────────────────

    describe("Column chooser hiding", () => {
        it("hides a default-visible column when toggled off", async () => {
            const user = userEvent.setup();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(headerCells().some((h) => h.textContent?.includes("Reason"))).toBe(true);

            await user.click(screen.getByRole("button", { name: "Columns" }));
            // "Reason" is also the visible column header, so target the checkbox
            // by role (the Menu dropdown is a11y-hidden in jsdom).
            await user.click(await screen.findByRole("checkbox", { name: "Reason", hidden: true }));

            await waitFor(() =>
                expect(headerCells().some((h) => h.textContent?.includes("Reason"))).toBe(false),
            );
        });

        it("keeps mandatory columns non-toggleable", async () => {
            const user = userEvent.setup();
            server.use(accessLogHandlers.list(createMockAccessLogResponse({ rows: [], total: 0 })));

            renderTable();

            await waitFor(
                () => expect(screen.getByText("No matching log entries.")).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByRole("button", { name: "Columns" }));

            const hostCheckbox = await screen.findByRole("checkbox", { name: "Host", hidden: true });
            expect(hostCheckbox).toBeDisabled();
            expect(hostCheckbox).toBeChecked();
        });
    });
});
