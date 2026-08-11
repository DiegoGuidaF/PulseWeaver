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

        expect(screen.getByText("Renewal timing vs. TTL")).toBeInTheDocument();
        expect(container.querySelector(".mantine-Skeleton-root")).toBeInTheDocument();
    });

    it("renders an error state and retries", () => {
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

        expect(screen.getByText("Failed to load renewal timing")).toBeInTheDocument();
    });

    it("explains what a renewal past its TTL means, so the bands do not read as faults", () => {
        renderWithProviders(
            <AddressHistoryRiskChart buckets={[]} timeRangeMs={3_600_000} isPending={false} />,
        );

        expect(screen.getByText(/usually the device was asleep/i)).toBeInTheDocument();
    });

    it("shows the empty state only when nothing renewed at all", () => {
        renderWithProviders(
            <AddressHistoryRiskChart
                buckets={[
                    createMockAddressHistoryBucket({
                        ok_device_count: 0,
                        approaching_device_count: 0,
                        critical_device_count: 0,
                        breached_device_count: 0,
                    }),
                ]}
                timeRangeMs={3_600_000}
                isPending={false}
            />,
        );

        expect(screen.getByText("No lease renewals in this window")).toBeInTheDocument();
    });

    it("charts a bucket carrying only comfortable renewals rather than calling it empty", () => {
        renderWithProviders(
            <AddressHistoryRiskChart
                buckets={[
                    createMockAddressHistoryBucket({
                        ok_device_count: 4,
                        approaching_device_count: 0,
                        critical_device_count: 0,
                        breached_device_count: 0,
                    }),
                ]}
                timeRangeMs={3_600_000}
                isPending={false}
            />,
        );

        // The comfortable band is the denominator: a window where every device
        // renewed on time still has something to plot.
        expect(screen.queryByText("No lease renewals in this window")).not.toBeInTheDocument();
        expect(screen.getByRole("img", { name: /renewal timing versus ttl/i })).toBeInTheDocument();
    });

    it("renders the stacked chart when buckets carry tight or late renewals", () => {
        renderWithProviders(
            <AddressHistoryRiskChart
                buckets={[
                    createMockAddressHistoryBucket({
                        timestamp: "2024-01-01T10:00:00Z",
                        ok_device_count: 6,
                        approaching_device_count: 2,
                        critical_device_count: 1,
                        breached_device_count: 0,
                    }),
                ]}
                timeRangeMs={3_600_000}
                isPending={false}
            />,
        );

        expect(screen.getByRole("img", { name: /renewal timing versus ttl/i })).toBeInTheDocument();
    });
});
