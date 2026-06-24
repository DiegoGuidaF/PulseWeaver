import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { IconPickerPopover } from "@/features/devices/IconPickerPopover";
import { renderWithProviders, setupUser } from "@/test/utils";

describe("IconPickerPopover", () => {
  it("marks the selected suggestion and calls select/close when another icon is chosen", async () => {
    const user = setupUser();
    const onSelect = vi.fn();
    const onClose = vi.fn();

    renderWithProviders(
      <IconPickerPopover
        opened
        selectedIcon="📡"
        deviceName="router"
        onSelect={onSelect}
        onClose={onClose}
        target={<button type="button">Choose icon</button>}
      />,
    );

    expect(screen.getByRole("button", { name: "📡" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "🔌" })).toHaveAttribute("aria-pressed", "false");

    await user.click(screen.getByRole("button", { name: "🔌" }));

    expect(onSelect).toHaveBeenCalledWith("🔌");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("accepts a custom icon value through the use action", async () => {
    const user = setupUser();
    const onSelect = vi.fn();

    renderWithProviders(
      <IconPickerPopover
        opened
        selectedIcon=""
        onSelect={onSelect}
        onClose={vi.fn()}
        target={<button type="button">Choose icon</button>}
      />,
    );

    await user.type(screen.getByPlaceholderText("paste emoji or URL…"), "🛰️");
    await user.click(screen.getByRole("button", { name: "use" }));

    expect(onSelect).toHaveBeenCalledWith("🛰️");
  });
});
