import { describe, expect, it } from "vitest";
import { deriveUserStatus } from "./userStatus";
import { createMockPolicyUserEntry, createMockPolicyUserIp } from "@/test/mocks/data";

describe("deriveUserStatus", () => {
  it("returns 'bypass' when bypass_allowlist is true regardless of IPs or grants", () => {
    const user = createMockPolicyUserEntry({
      bypass_allowlist: true,
      ips: [],
      allowed_host_count: 0,
    });
    expect(deriveUserStatus(user)).toBe("bypass");
  });

  it("returns 'bypass' even when the bypass user has live IPs", () => {
    const user = createMockPolicyUserEntry({
      bypass_allowlist: true,
      ips: [createMockPolicyUserIp()],
      allowed_host_count: 3,
    });
    expect(deriveUserStatus(user)).toBe("bypass");
  });

  it("returns 'live_with_access' when user has live IPs and host grants", () => {
    const user = createMockPolicyUserEntry({
      bypass_allowlist: false,
      ips: [createMockPolicyUserIp()],
      allowed_host_count: 2,
    });
    expect(deriveUserStatus(user)).toBe("live_with_access");
  });

  it("returns 'live_no_host_access' when user has live IPs but no host grants", () => {
    // This is the key bug case: device is active but allowlist is empty (after revoke).
    const user = createMockPolicyUserEntry({
      bypass_allowlist: false,
      ips: [createMockPolicyUserIp()],
      allowed_host_count: 0,
    });
    expect(deriveUserStatus(user)).toBe("live_no_host_access");
  });

  it("returns 'no_live_ips' when user has no live IPs but has host grants", () => {
    const user = createMockPolicyUserEntry({
      bypass_allowlist: false,
      ips: [],
      allowed_host_count: 4,
    });
    expect(deriveUserStatus(user)).toBe("no_live_ips");
  });

  it("returns 'no_access' when user has no live IPs and no host grants", () => {
    const user = createMockPolicyUserEntry({
      bypass_allowlist: false,
      ips: [],
      allowed_host_count: 0,
    });
    expect(deriveUserStatus(user)).toBe("no_access");
  });
});
