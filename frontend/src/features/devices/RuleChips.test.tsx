import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { RuleChips } from "@/features/devices/RuleChips";
import { DeviceState, type DeviceListEntry } from "@/lib/api";
import { renderWithProviders } from "@/test/utils";

function entry(overrides?: Partial<DeviceListEntry>): DeviceListEntry {
  return {
    id: 1,
    name: "Router",
    state: DeviceState.HEALTHY,
    live_address_count: 2,
    rules: [],
    created_at: "2026-06-01T00:00:00Z",
    ...overrides,
  };
}

describe("RuleChips", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-06-01T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders enabled rule labels and omits disabled or absent rules", () => {
    renderWithProviders(
      <RuleChips
        entry={entry({
          live_address_count: 3,
          rules: [
            { type: "auto_expiry", enabled: true, ttl_seconds: 3600 },
            { type: "max_active", enabled: true, limit: 3 },
            { type: "auto_expiry", enabled: false, ttl_seconds: 60 },
          ],
        })}
      />,
    );

    expect(screen.getByLabelText("Auto-expiry · TTL 1h")).toHaveTextContent("1h");
    expect(
      screen.getByLabelText("Max active IPs · at limit (3/3) · next IP will evict oldest"),
    ).toHaveTextContent("3/3");
    expect(screen.queryByText("1m")).not.toBeInTheDocument();
  });

  it("renders no rule chips when only disabled rules are present", () => {
    const { container } = renderWithProviders(
      <RuleChips
        entry={entry({
          rules: [
            { type: "auto_expiry", enabled: false, ttl_seconds: 3600 },
            { type: "max_active", enabled: false, limit: 2 },
          ],
        })}
      />,
    );

    expect(container.querySelector(".mantine-Badge-root")).not.toBeInTheDocument();
  });

  it("renders pending and recently expired pairing labels", () => {
    const pending = renderWithProviders(
      <RuleChips
        entry={entry({
          pairing: {
            status: "pending",
            expires_at: "2026-06-01T12:45:00Z",
            updated_at: "2026-06-01T12:00:00Z",
          },
        })}
      />,
    );

    expect(screen.getByLabelText("Pairing pending · 45m left")).toHaveTextContent("45m left");
    pending.unmount();

    renderWithProviders(
      <RuleChips
        entry={entry({
          pairing: {
            status: "expired",
            expires_at: "2026-05-31T12:00:00Z",
            updated_at: "2026-05-31T12:00:00Z",
          },
        })}
      />,
    );

    expect(screen.getByLabelText("Pairing code expired — regenerate required")).toHaveTextContent(
      "expired",
    );
  });
});
