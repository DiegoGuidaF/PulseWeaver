import { describe, expect, it } from "vitest";
import { QueryClient } from "@tanstack/react-query";
import type { OwnerFleetGroup } from "@/lib/api";
import { createMockFleetDevice, createMockOwnerFleetGroup, createMockOwnerSummary } from "@/test/mocks/data";
import {
  fleetListKey,
  fleetOwnerKey,
  invalidateFleetList,
  invalidateOwnerFleet,
  invalidateOwnerIdentity,
  spliceOwnerGroup,
} from "@/features/devices/fleetCache";

function group(id: number, deviceName: string): OwnerFleetGroup {
  return createMockOwnerFleetGroup({
    owner: createMockOwnerSummary({ id, display_name: `Owner ${id}` }),
    devices: [createMockFleetDevice({ id, name: deviceName })],
  });
}

describe("spliceOwnerGroup", () => {
  it("leaves a cold list cache untouched", () => {
    const queryClient = new QueryClient();

    spliceOwnerGroup(queryClient, group(1, "Work Laptop"));

    // Seeding the key here would leave the list page rendering one owner as the
    // whole fleet.
    expect(queryClient.getQueryData(fleetListKey())).toBeUndefined();
  });

  it("replaces only the matching owner's group in a cached list", () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(fleetListKey(), [group(1, "Old"), group(2, "Other")]);

    spliceOwnerGroup(queryClient, group(1, "Renamed"));

    const cached = queryClient.getQueryData<OwnerFleetGroup[]>(fleetListKey());
    expect(cached?.map((g) => g.devices[0].name)).toEqual(["Renamed", "Other"]);
  });

  it("keeps the list's own freshness timestamp", () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(fleetListKey(), [group(1, "Old")], { updatedAt: 1_000 });

    spliceOwnerGroup(queryClient, group(1, "Renamed"));

    expect(queryClient.getQueryState(fleetListKey())?.dataUpdatedAt).toBe(1_000);
  });
});

describe("fleet invalidation scope", () => {
  function seeded() {
    const queryClient = new QueryClient();
    queryClient.setQueryData(fleetListKey(), [group(1, "A"), group(2, "B")]);
    queryClient.setQueryData(fleetOwnerKey(1), [group(1, "A")]);
    queryClient.setQueryData(fleetOwnerKey(2), [group(2, "B")]);
    return queryClient;
  }

  const invalidated = (c: QueryClient, key: ReturnType<typeof fleetListKey>) =>
    c.getQueryState(key)?.isInvalidated === true;

  it("invalidates one owner without touching the list or its siblings", () => {
    const queryClient = seeded();

    invalidateOwnerFleet(queryClient, 1);

    expect(invalidated(queryClient, fleetOwnerKey(1))).toBe(true);
    expect(invalidated(queryClient, fleetOwnerKey(2))).toBe(false);
    expect(invalidated(queryClient, fleetListKey())).toBe(false);
  });

  // The list key is an owner key minus `query`, so it partial-matches every owner
  // entry. Dropping `exact` here turns one refetch into one per cached owner.
  it("invalidates the list without dragging any owner entry along", () => {
    const queryClient = seeded();

    invalidateFleetList(queryClient);

    expect(invalidated(queryClient, fleetListKey())).toBe(true);
    expect(invalidated(queryClient, fleetOwnerKey(1))).toBe(false);
    expect(invalidated(queryClient, fleetOwnerKey(2))).toBe(false);
  });

  it("sweeps the list and every owner entry when owner identity changes", () => {
    const queryClient = seeded();

    invalidateOwnerIdentity(queryClient);

    expect(invalidated(queryClient, fleetListKey())).toBe(true);
    expect(invalidated(queryClient, fleetOwnerKey(1))).toBe(true);
    expect(invalidated(queryClient, fleetOwnerKey(2))).toBe(true);
  });
});
