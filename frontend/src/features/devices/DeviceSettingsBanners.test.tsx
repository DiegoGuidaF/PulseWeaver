import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { DeviceApiKeyRuleHintBanner } from "@/features/devices/DeviceApiKeyRuleHintBanner";
import { DeviceDisabledBanner } from "@/features/devices/DeviceDisabledBanner";
import { TEST_TIMEOUTS } from "@/test/constants";
import { renderWithProviders, setupUser } from "@/test/utils";

describe("DeviceApiKeyRuleHintBanner", () => {
  it("renders API-key-without-limits guidance and action callbacks", async () => {
    const user = setupUser();
    const onGoToRules = vi.fn();
    const onGoToSettings = vi.fn();

    renderWithProviders(
      <DeviceApiKeyRuleHintBanner
        deviceId={1}
        onGoToRules={onGoToRules}
        onGoToSettings={onGoToSettings}
      />,
    );

    expect(screen.getByText("API key active with no address limits")).toBeInTheDocument();
    expect(screen.getByText(/old IPs accumulate indefinitely/i)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Configure limits →" }));
    await user.click(screen.getByRole("button", { name: "Remove API key →" }));

    expect(onGoToRules).toHaveBeenCalledTimes(1);
    expect(onGoToSettings).toHaveBeenCalledTimes(1);
  });

  it("can be dismissed", async () => {
    const user = setupUser();

    renderWithProviders(
      <DeviceApiKeyRuleHintBanner
        deviceId={2}
        onGoToRules={vi.fn()}
        onGoToSettings={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Dismiss" }));

    expect(screen.queryByText("API key active with no address limits")).not.toBeInTheDocument();
  });
});

describe("DeviceDisabledBanner", () => {
  it("renders the frozen-device warning and re-enable action", async () => {
    const user = setupUser();

    renderWithProviders(<DeviceDisabledBanner deviceId={1} />);

    expect(screen.getByText("Device frozen")).toBeInTheDocument();
    expect(screen.getByText("Address updates are blocked until re-enabled.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Re-enable" }));

    await waitFor(
      () => {
        expect(
          screen.getByText("Device re-enabled — address updates are allowed again."),
        ).toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );
  });
});
