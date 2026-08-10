import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { AddressHistoryAtRiskStrip } from "./AddressHistoryAtRiskStrip";
import { createMockAddressHistoryAtRiskDevice } from "@/test/mocks/data";
import { renderWithProviders, setupUser } from "@/test/utils";
import { TtlRisk } from "@/lib/api";

describe("AddressHistoryAtRiskStrip", () => {
    it("renders nothing when there are no at-risk devices", () => {
        renderWithProviders(<AddressHistoryAtRiskStrip devices={[]} onSelectDevice={vi.fn()} />);

        expect(screen.queryByText("At-risk devices")).not.toBeInTheDocument();
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("renders each device's worst risk, name, and total event count", () => {
        renderWithProviders(
            <AddressHistoryAtRiskStrip
                devices={[
                    createMockAddressHistoryAtRiskDevice({
                        device_id: 1,
                        device_name: "nas-01",
                        worst_risk: TtlRisk.BREACHED,
                        event_count: 14,
                    }),
                ]}
                onSelectDevice={vi.fn()}
            />,
        );

        expect(screen.getByText("nas-01")).toBeInTheDocument();
        expect(screen.getByText("Breached")).toBeInTheDocument();
        // The count is every event in the window, not a breach tally — it must
        // read as an event count, never bare beside the risk badge as if it
        // were "14 breaches".
        expect(screen.getByText("14 events")).toBeInTheDocument();
    });

    it("applies the device_id filter when an entry is clicked", async () => {
        const user = setupUser();
        const onSelectDevice = vi.fn();
        renderWithProviders(
            <AddressHistoryAtRiskStrip
                devices={[createMockAddressHistoryAtRiskDevice({ device_id: 42, device_name: "nas-01" })]}
                onSelectDevice={onSelectDevice}
            />,
        );

        await user.click(screen.getByRole("button", { name: "Filter by nas-01" }));

        expect(onSelectDevice).toHaveBeenCalledWith(42);
    });
});
