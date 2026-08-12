import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { AddressHistoryPage } from "@/pages/address-history/AddressHistoryPage";
import {
    createMockAddressHistoryResponse,
    createMockAddressHistoryTuningCandidate,
    createMockAddressHistoryTuningResponse,
} from "@/test/mocks/data";
import { TEST_TIMEOUTS } from "@/test/constants";
import { addressHandlers, endpoints } from "@/test/mocks/handlers";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";

const BASE_ENTRY =
    "/address-history?from=2024-01-01T00%3A00%3A00.000Z&to=2024-01-02T00%3A00%3A00.000Z";

/**
 * Mounted through the whole page rather than in isolation, because what is under
 * test is the boundary between the window-scoped readout and the filtered table.
 */
function renderPage() {
    return renderWithProviders(<AddressHistoryPage />, { initialEntries: [BASE_ENTRY] });
}

describe("AddressHistoryTuningSection", () => {
    it("ranks devices and applies the device_id filter on click", async () => {
        const eventsQueries: string[] = [];
        server.use(
            http.get(endpoints.addressHistory, ({ request }) => {
                eventsQueries.push(new URL(request.url).search);
                return HttpResponse.json(createMockAddressHistoryResponse());
            }),
            addressHandlers.tuning.success(
                createMockAddressHistoryTuningResponse({
                    devices: [
                        createMockAddressHistoryTuningCandidate({
                            device_id: 7,
                            device_name: "nas-01",
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
        renderPage();

        await waitFor(() => expect(screen.getByText("nas-01")).toBeInTheDocument(), {
            timeout: TEST_TIMEOUTS.SHORT,
        });
        expect(screen.getByText(/189 renewals · 6 past TTL \(3%\)/)).toBeInTheDocument();

        await user.click(screen.getByRole("button", { name: "Filter by nas-01" }));

        await waitFor(() => expect(eventsQueries.at(-1)).toContain("device_id=7"), {
            timeout: TEST_TIMEOUTS.SHORT,
        });
    });

    it("never sends the table's column filters to the tuning endpoint", async () => {
        const tuningQueries: string[] = [];
        const eventsQueries: string[] = [];
        server.use(
            http.get(endpoints.addressHistoryTuning, ({ request }) => {
                tuningQueries.push(new URL(request.url).search);
                return HttpResponse.json(
                    createMockAddressHistoryTuningResponse({
                        devices: [
                            createMockAddressHistoryTuningCandidate({ device_id: 7, device_name: "nas-01" }),
                        ],
                    }),
                );
            }),
            http.get(endpoints.addressHistory, ({ request }) => {
                eventsQueries.push(new URL(request.url).search);
                return HttpResponse.json(createMockAddressHistoryResponse());
            }),
        );

        const user = setupUser();
        renderPage();

        await waitFor(() => expect(screen.getByText("nas-01")).toBeInTheDocument(), {
            timeout: TEST_TIMEOUTS.SHORT,
        });

        // The readout's own device link filters the events table below it —
        // which is exactly the interaction that used to re-rank the fleet.
        await user.click(screen.getByRole("button", { name: "Filter by nas-01" }));

        await waitFor(() => expect(eventsQueries.at(-1)).toContain("device_id=7"), {
            timeout: TEST_TIMEOUTS.SHORT,
        });
        expect(tuningQueries.every((q) => !q.includes("device_id"))).toBe(true);
    });

    it("reports an empty ranking as a reading", async () => {
        server.use(addressHandlers.tuning.empty());

        renderPage();

        await waitFor(
            () =>
                expect(
                    screen.getByText("No device's TTL needs resizing in this window."),
                ).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });
});
