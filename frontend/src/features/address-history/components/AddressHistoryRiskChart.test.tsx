import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { AddressHistoryRiskChart } from "./AddressHistoryRiskChart";
import { createMockAddressHistoryBucket } from "@/test/mocks/data";
import { renderWithProviders } from "@/test/utils";

describe("AddressHistoryRiskChart", () => {
    it("renders a loading skeleton while pending", () => {
        const { container } = renderWithProviders(
            <AddressHistoryRiskChart buckets={undefined} timeRangeMs={3_600_000} isPending />,
        );

        expect(screen.getByText("Devices at risk over time")).toBeInTheDocument();
        expect(container.querySelector(".mantine-Skeleton-root")).toBeInTheDocument();
    });

    it("renders an error state and retries", async () => {
        const onRetry = vi.fn();
        renderWithProviders(
            <AddressHistoryRiskChart
                buckets={undefined}
                timeRangeMs={3_600_000}
                isPending={false}
                error={new Error("boom")}
                onRetry={onRetry}
            />,
        );

        expect(screen.getByText("Failed to load risk history")).toBeInTheDocument();
    });

    it("shows the reassuring empty state when no buckets carry at-risk devices", () => {
        renderWithProviders(
            <AddressHistoryRiskChart buckets={[]} timeRangeMs={3_600_000} isPending={false} />,
        );

        expect(screen.getByText("No devices at risk in this period")).toBeInTheDocument();
    });

    it("renders the stacked chart when buckets carry at-risk devices", () => {
        renderWithProviders(
            <AddressHistoryRiskChart
                buckets={[
                    createMockAddressHistoryBucket({
                        timestamp: "2024-01-01T10:00:00Z",
                        approaching_device_count: 2,
                        critical_device_count: 1,
                        breached_device_count: 0,
                    }),
                ]}
                timeRangeMs={3_600_000}
                isPending={false}
            />,
        );

        expect(screen.queryByText("No devices at risk in this period")).not.toBeInTheDocument();
        expect(
            screen.getByRole("img", { name: /devices at risk over time: stacked bar chart/i }),
        ).toBeInTheDocument();
    });
});
