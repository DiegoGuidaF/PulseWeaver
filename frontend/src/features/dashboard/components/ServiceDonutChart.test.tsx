import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { ServiceDonutChart } from "./ServiceDonutChart";
import { createMockDashboardServiceCount } from "@/test/mocks/data";
import { renderWithProviders } from "@/test/utils";

describe("ServiceDonutChart", () => {
    it("renders the loading branch", () => {
        const { container } = renderWithProviders(
            <ServiceDonutChart data={undefined} isLoading />,
        );

        expect(screen.getByText("Requests by Service")).toBeInTheDocument();
        expect(container.querySelector(".mantine-Skeleton-root")).toBeInTheDocument();
    });

    it("renders the empty branch", () => {
        renderWithProviders(<ServiceDonutChart data={[]} isLoading={false} />);

        expect(screen.getByText("No service data for this period.")).toBeInTheDocument();
    });

    it("renders service labels and chart accessibility text for data", () => {
        renderWithProviders(
            <ServiceDonutChart
                isLoading={false}
                data={[
                    createMockDashboardServiceCount({
                        host: "app.example.com",
                        allow_count: 80,
                        deny_count: 20,
                    }),
                    createMockDashboardServiceCount({
                        host: "",
                        allow_count: 5,
                        deny_count: 1,
                    }),
                ]}
            />,
        );

        expect(screen.getByText("app.example.com")).toBeInTheDocument();
        expect(screen.getByText("(unknown)")).toBeInTheDocument();
        expect(screen.getByRole("img", { name: /requests by service: donut chart/i })).toBeInTheDocument();
    });
});
