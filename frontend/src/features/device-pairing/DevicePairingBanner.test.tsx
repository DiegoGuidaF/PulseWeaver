import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { DevicePairingBanner } from "@/features/device-pairing/DevicePairingBanner";
import { renderWithProviders, setupUser } from "@/test/utils";

describe("DevicePairingBanner", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the outstanding-code warning and view action", async () => {
    const user = setupUser();
    const onViewPairing = vi.fn();

    renderWithProviders(
      <DevicePairingBanner
        expiresAt={new Date(Date.now() + 46 * 60 * 1000).toISOString()}
        onViewPairing={onViewPairing}
      />,
    );

    expect(screen.getByText("Pairing code outstanding")).toBeInTheDocument();
    expect(screen.getByText(/m remaining until expiry/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "View pairing →" }));

    expect(onViewPairing).toHaveBeenCalledTimes(1);
  });

  it("renders expired, minute, and hour expiry branches", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-06-01T12:00:00Z"));

    const expired = renderWithProviders(
      <DevicePairingBanner expiresAt="2026-06-01T11:59:00Z" onViewPairing={vi.fn()} />,
    );

    expect(screen.getByText("expired until expiry")).toBeInTheDocument();
    expired.unmount();

    const minutes = renderWithProviders(
      <DevicePairingBanner expiresAt="2026-06-01T12:30:00Z" onViewPairing={vi.fn()} />,
    );
    expect(screen.getByText("30m remaining until expiry")).toBeInTheDocument();
    minutes.unmount();

    renderWithProviders(
      <DevicePairingBanner expiresAt="2026-06-01T14:00:00Z" onViewPairing={vi.fn()} />,
    );
    expect(screen.getByText("2h remaining until expiry")).toBeInTheDocument();
  });
});
