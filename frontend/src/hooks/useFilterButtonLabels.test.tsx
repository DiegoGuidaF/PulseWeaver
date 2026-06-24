import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { useFilterButtonLabels } from "./useFilterButtonLabels";
import { renderWithProviders } from "@/test/utils";

function FilterHeaderFixture({ labels }: { labels: Record<string, string> }) {
    const ref = useFilterButtonLabels(labels);
    return (
        <div ref={ref}>
            <table>
                <thead>
                    <tr>
                        <th data-accessor="outcome">
                            <button aria-haspopup="dialog">Outcome filter</button>
                        </th>
                        <th data-accessor="reason">
                            <button aria-haspopup="dialog" aria-label="Existing label">
                                Reason filter
                            </button>
                        </th>
                        <th data-accessor="client_ip">
                            <button aria-haspopup="dialog">Client IP filter</button>
                        </th>
                    </tr>
                </thead>
            </table>
        </div>
    );
}

describe("useFilterButtonLabels", () => {
    it("adds labels from the accessor map and leaves unmapped filters unlabelled", () => {
        renderWithProviders(
            <FilterHeaderFixture labels={{ outcome: "Filter by outcome" }} />,
        );

        expect(screen.getByRole("button", { name: "Filter by outcome" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Client IP filter" })).not.toHaveAttribute("aria-label");
    });

    it("preserves an existing aria-label while adding composed labels", () => {
        renderWithProviders(
            <FilterHeaderFixture
                labels={{
                    outcome: "Filter by outcome",
                    reason: "Filter by denial reason",
                    client_ip: "Filter by client IP",
                }}
            />,
        );

        expect(screen.getByRole("button", { name: "Existing label" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Filter by client IP" })).toBeInTheDocument();
    });
});
