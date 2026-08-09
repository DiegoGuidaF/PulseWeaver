import { afterEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import { PairingCodeDisplay } from "@/features/device-pairing/PairingCodeDisplay";
import { TEST_TIMEOUTS } from "@/test/constants";
import { createMockDevicePairing } from "@/test/mocks/data";
import { renderWithProviders, setupUser } from "@/test/utils";

function setClipboard(writeText: (text: string) => Promise<void>) {
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText },
    configurable: true,
  });
}

describe("PairingCodeDisplay", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("displays the pairing code and copies it for the user", async () => {
    const user = setupUser();
    const writeText = vi.fn().mockResolvedValue(undefined);
    setClipboard(writeText);

    renderWithProviders(
      <PairingCodeDisplay
        deviceId={1}
        ownerId={1}
        pairing={createMockDevicePairing({ pairing_code: "PW-ABCD-1234" })}
        onRevoke={vi.fn()}
      />,
    );

    expect(screen.getByText("PW-ABCD-1234")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Copy code" }));

    await waitFor(
      () => {
        expect(writeText).toHaveBeenCalledWith("PW-ABCD-1234");
        expect(screen.getByText("Pairing code copied")).toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );
  });

  it("enlarges the QR code for scanning straight off the screen", async () => {
    const user = setupUser();

    renderWithProviders(
      <PairingCodeDisplay
        deviceId={1}
        ownerId={1}
        pairing={createMockDevicePairing({ pairing_code: "PW-ABCD-1234" })}
        onRevoke={vi.fn()}
      />,
    );

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Enlarge QR code" }));

    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByTitle("Pairing code QR")).toBeInTheDocument();
    expect(within(dialog).getByText("PW-ABCD-1234")).toBeInTheDocument();
  });

  it("copies the QR to the clipboard as an image", async () => {
    const user = setupUser();
    const write = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { write }, configurable: true });
    vi.stubGlobal(
      "ClipboardItem",
      class {
        items: Record<string, Promise<Blob>>;
        constructor(items: Record<string, Promise<Blob>>) {
          this.items = items;
        }
      },
    );

    renderWithProviders(
      <PairingCodeDisplay
        deviceId={1}
        ownerId={1}
        pairing={createMockDevicePairing()}
        onRevoke={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: /copy qr/i }));

    await waitFor(
      () => {
        expect(write).toHaveBeenCalledTimes(1);
        expect(Object.keys(write.mock.calls[0][0][0].items)).toEqual(["image/png"]);
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );
  });

  it("shows the repair reassurance copy", () => {
    renderWithProviders(
      <PairingCodeDisplay
        deviceId={1}
        ownerId={1}
        pairing={createMockDevicePairing()}
        onRevoke={vi.fn()}
        isRepair
      />,
    );

    expect(screen.getByText(/current link stays active/i)).toBeInTheDocument();
  });

  it("renders expired, minute, and hour expiry branches", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-06-01T12:00:00Z"));

    const expired = renderWithProviders(
      <PairingCodeDisplay
        deviceId={1}
        ownerId={1}
        pairing={createMockDevicePairing({ expires_at: "2026-06-01T11:59:00Z" })}
        onRevoke={vi.fn()}
      />,
    );

    expect(screen.getByText("expired")).toBeInTheDocument();
    expired.unmount();

    const minutes = renderWithProviders(
      <PairingCodeDisplay
        deviceId={1}
        ownerId={1}
        pairing={createMockDevicePairing({ expires_at: "2026-06-01T12:45:00Z" })}
        onRevoke={vi.fn()}
      />,
    );
    expect(screen.getByText("45m remaining")).toBeInTheDocument();
    minutes.unmount();

    renderWithProviders(
      <PairingCodeDisplay
        deviceId={1}
        ownerId={1}
        pairing={createMockDevicePairing({ expires_at: "2026-06-01T14:15:00Z" })}
        onRevoke={vi.fn()}
      />,
    );
    expect(screen.getByText("2h 15m remaining")).toBeInTheDocument();
  });
});
