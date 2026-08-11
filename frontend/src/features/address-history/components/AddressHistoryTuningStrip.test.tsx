import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { AddressHistoryTuningStrip } from "./AddressHistoryTuningStrip";
import { createMockAddressHistoryAtRiskDevice } from "@/test/mocks/data";
import { renderWithProviders, setupUser } from "@/test/utils";
import { TtlRisk } from "@/lib/api";

describe("AddressHistoryTuningStrip", () => {
    it("renders nothing when no device needs tuning", () => {
        renderWithProviders(
            <AddressHistoryTuningStrip devices={[]} onSelectDevice={vi.fn()} onTuneDevice={vi.fn()} />,
        );

        expect(screen.queryByText("Devices worth tuning")).not.toBeInTheDocument();
    });

    it("reads out the numbers needed to resize the TTL", () => {
        renderWithProviders(
            <AddressHistoryTuningStrip
                devices={[
                    createMockAddressHistoryAtRiskDevice({
                        device_name: "nas-01",
                        worst_risk: TtlRisk.BREACHED,
                        renewal_count: 189,
                        late_renewal_count: 6,
                        ttl_seconds: 3600,
                        p95_gap_seconds: 4320,
                    }),
                ]}
                onSelectDevice={vi.fn()}
                onTuneDevice={vi.fn()}
            />,
        );

        expect(screen.getByText("nas-01")).toBeInTheDocument();
        expect(screen.getByText("Past TTL")).toBeInTheDocument();
        expect(
            screen.getByText(/TTL 1h · 189 renewals · 6 past TTL \(3%\) · p95 gap 1h 12m/),
        ).toBeInTheDocument();
    });

    it("suggests the lowest preset that would cover the p95 gap", async () => {
        const user = setupUser();
        const onTuneDevice = vi.fn();
        renderWithProviders(
            <AddressHistoryTuningStrip
                devices={[
                    createMockAddressHistoryAtRiskDevice({
                        device_id: 7,
                        ttl_seconds: 3600,
                        p95_gap_seconds: 4320,
                    }),
                ]}
                onSelectDevice={vi.fn()}
                onTuneDevice={onTuneDevice}
            />,
        );

        // 4320s clears 1h but not 6h, so the ladder's next rung is the answer.
        await user.click(screen.getByRole("button", { name: "Set TTL 6h" }));

        expect(onTuneDevice).toHaveBeenCalledWith(7);
    });

    it("offers a custom TTL when the p95 gap exceeds the top preset", () => {
        renderWithProviders(
            <AddressHistoryTuningStrip
                devices={[
                    createMockAddressHistoryAtRiskDevice({ ttl_seconds: 3600, p95_gap_seconds: 200_000 }),
                ]}
                onSelectDevice={vi.fn()}
                onTuneDevice={vi.fn()}
            />,
        );

        expect(screen.getByRole("button", { name: "Set a custom TTL" })).toBeInTheDocument();
    });

    it("suggests nothing when the current TTL already covers the p95 gap", () => {
        renderWithProviders(
            <AddressHistoryTuningStrip
                devices={[
                    createMockAddressHistoryAtRiskDevice({ ttl_seconds: 21600, p95_gap_seconds: 4320 }),
                ]}
                onSelectDevice={vi.fn()}
                onTuneDevice={vi.fn()}
            />,
        );

        expect(screen.queryByRole("button", { name: /Set TTL/ })).not.toBeInTheDocument();
    });

    it("applies the device filter when the name is clicked", async () => {
        const user = setupUser();
        const onSelectDevice = vi.fn();
        renderWithProviders(
            <AddressHistoryTuningStrip
                devices={[createMockAddressHistoryAtRiskDevice({ device_id: 42, device_name: "nas-01" })]}
                onSelectDevice={onSelectDevice}
                onTuneDevice={vi.fn()}
            />,
        );

        await user.click(screen.getByRole("button", { name: "Filter by nas-01" }));

        expect(onSelectDevice).toHaveBeenCalledWith(42);
    });

    it("still reads out the device it is scoped to, without the redundant filter link", () => {
        renderWithProviders(
            <AddressHistoryTuningStrip
                devices={[createMockAddressHistoryAtRiskDevice({ device_id: 42, device_name: "nas-01" })]}
                onSelectDevice={vi.fn()}
                onTuneDevice={vi.fn()}
                deviceScoped
            />,
        );

        expect(screen.getByText("TTL tuning")).toBeInTheDocument();
        expect(screen.getByText("nas-01")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Filter by nas-01" })).not.toBeInTheDocument();
    });
});
