import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { AddressHistoryTuningStrip } from "./AddressHistoryTuningStrip";
import { createMockAddressHistoryTuningCandidate } from "@/test/mocks/data";
import { renderWithProviders, setupUser } from "@/test/utils";

/** Every prop the strip needs, so each test states only what it is about. */
function renderStrip(props: Partial<Parameters<typeof AddressHistoryTuningStrip>[0]> = {}) {
    return renderWithProviders(
        <AddressHistoryTuningStrip
            devices={[createMockAddressHistoryTuningCandidate()]}
            total={1}
            minRenewals={10}
            scopeLabel="all devices · last 1w"
            onSelectDevice={vi.fn()}
            onTuneDevice={vi.fn()}
            isPending={false}
            {...props}
        />,
    );
}

describe("AddressHistoryTuningStrip", () => {
    it("says so when no device needs tuning, rather than vanishing", () => {
        renderStrip({ devices: [], total: 0 });

        expect(screen.getByText("Devices worth tuning")).toBeInTheDocument();
        expect(screen.getByText("No device's TTL needs resizing in this window.")).toBeInTheDocument();
    });

    it("states the window it was computed over", () => {
        renderStrip({ scopeLabel: "all devices · last 1w" });

        expect(screen.getByText("all devices · last 1w")).toBeInTheDocument();
    });

    it("discloses the sample floor it filtered on", () => {
        renderStrip({ minRenewals: 10 });

        expect(
            screen.getByText(/at least 10 renewals whose TTL misses more than 5% of them/),
        ).toBeInTheDocument();
    });

    it("says how many devices the shown few were drawn from", () => {
        renderStrip({ total: 9 });

        expect(screen.getByText(/Showing the 1 widest misses of 9 devices/)).toBeInTheDocument();
    });

    it("reads out the numbers needed to resize the TTL", () => {
        renderStrip({
            devices: [
                createMockAddressHistoryTuningCandidate({
                    device_name: "nas-01",
                    renewal_count: 189,
                    late_renewal_count: 6,
                    ttl_seconds: 3600,
                    p95_gap_seconds: 4320,
                }),
            ],
        });

        expect(screen.getByText("nas-01")).toBeInTheDocument();
        expect(
            screen.getByText(/TTL 1h · 189 renewals · 6 past TTL \(3%\) · p95 gap 1h 12m/),
        ).toBeInTheDocument();
    });

    it("badges the size of the miss, not the fact of it", () => {
        renderStrip({
            devices: [createMockAddressHistoryTuningCandidate({ ttl_seconds: 3600, p95_gap_seconds: 4320 })],
        });

        expect(screen.getByText("1h → 6h")).toBeInTheDocument();
    });

    it("suggests the lowest preset that would cover the p95 gap", async () => {
        const user = setupUser();
        const onTuneDevice = vi.fn();
        renderStrip({
            devices: [
                createMockAddressHistoryTuningCandidate({
                    device_id: 7,
                    ttl_seconds: 3600,
                    p95_gap_seconds: 4320,
                }),
            ],
            onTuneDevice,
        });

        // 4320s clears 1h but not 6h, so the ladder's next rung is the answer.
        await user.click(screen.getByRole("button", { name: "Set TTL 6h" }));

        expect(onTuneDevice).toHaveBeenCalledWith(7, 21600);
    });

    it("hands over the raw gap when it exceeds the top preset", async () => {
        const user = setupUser();
        const onTuneDevice = vi.fn();
        renderStrip({
            devices: [
                createMockAddressHistoryTuningCandidate({
                    device_id: 7,
                    ttl_seconds: 3600,
                    p95_gap_seconds: 200_000,
                }),
            ],
            onTuneDevice,
        });

        // No rung covers the gap, so the measurement travels instead and the
        // control falls back to its custom input.
        await user.click(screen.getByRole("button", { name: "Set a custom TTL" }));

        expect(onTuneDevice).toHaveBeenCalledWith(7, 200_000);
    });

    it("reads out a comfortable TTL without offering to change it", () => {
        // Reachable device-scoped, where the readout skips the selection rule.
        renderStrip({
            devices: [
                createMockAddressHistoryTuningCandidate({ ttl_seconds: 21600, p95_gap_seconds: 4320 }),
            ],
            deviceScoped: true,
        });

        expect(screen.getByText("Comfortable")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Set TTL/ })).not.toBeInTheDocument();
    });

    it("applies the device filter when the name is clicked", async () => {
        const user = setupUser();
        const onSelectDevice = vi.fn();
        renderStrip({
            devices: [
                createMockAddressHistoryTuningCandidate({ device_id: 42, device_name: "nas-01" }),
            ],
            onSelectDevice,
        });

        await user.click(screen.getByRole("button", { name: "Filter by nas-01" }));

        expect(onSelectDevice).toHaveBeenCalledWith(42);
    });

    it("still reads out the device it is scoped to, without the redundant filter link", () => {
        renderStrip({
            devices: [
                createMockAddressHistoryTuningCandidate({ device_id: 42, device_name: "nas-01" }),
            ],
            deviceScoped: true,
        });

        expect(screen.getByText("TTL tuning")).toBeInTheDocument();
        expect(screen.getByText("nas-01")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Filter by nas-01" })).not.toBeInTheDocument();
    });

    it("surfaces a load failure with a retry", () => {
        const onRetry = vi.fn();
        renderStrip({ error: new Error("boom"), onRetry });

        expect(screen.getByText("Failed to load TTL tuning")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument();
    });
});
