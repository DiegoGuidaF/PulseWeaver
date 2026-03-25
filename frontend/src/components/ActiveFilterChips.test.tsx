import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/utils";
import { ActiveFilterChips, type FilterChip } from "./ActiveFilterChips";

function renderChips(chips: FilterChip[]) {
    return renderWithProviders(<ActiveFilterChips chips={chips} />);
}

/** Mantine Pill's remove button is aria-hidden, so we find it via class. */
function getRemoveButtons(container: HTMLElement) {
    return container.querySelectorAll(".mantine-Pill-remove");
}

describe("ActiveFilterChips", () => {
    it("renders nothing when chips array is empty", () => {
        const { container } = renderChips([]);
        expect(container.querySelector(".mantine-Pill-root")).toBeNull();
    });

    it("renders a pill for each chip with label and value", () => {
        renderChips([
            { label: "IP", value: "192.168.1.1", onRemove: vi.fn() },
            { label: "Device", value: "Router", onRemove: vi.fn() },
        ]);

        expect(screen.getByText("IP:")).toBeInTheDocument();
        expect(screen.getByText(/192\.168\.1\.1/)).toBeInTheDocument();
        expect(screen.getByText("Device:")).toBeInTheDocument();
        expect(screen.getByText(/Router/)).toBeInTheDocument();
    });

    it("calls onRemove when the remove button is clicked", async () => {
        const user = userEvent.setup();
        const onRemove = vi.fn();

        const { container } = renderChips([
            { label: "IP", value: "10.0.0.1", onRemove },
        ]);

        const removeBtn = getRemoveButtons(container)[0] as HTMLElement;
        expect(removeBtn).toBeDefined();
        await user.click(removeBtn);

        expect(onRemove).toHaveBeenCalledOnce();
    });

    it("renders a remove button for each chip", () => {
        const { container } = renderChips([
            { label: "IP", value: "10.0.0.1", onRemove: vi.fn() },
            { label: "Outcome", value: "Deny", onRemove: vi.fn() },
            { label: "Reason", value: "No matching device", onRemove: vi.fn() },
        ]);

        expect(getRemoveButtons(container)).toHaveLength(3);
    });
});
