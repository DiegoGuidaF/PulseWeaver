import { describe, expect, it } from "vitest";
import type { KnownHostWithStats } from "@/lib/api";
import {
  diffHosts,
  fromServerHosts,
  hostsDraftReducer,
  isDirtyHosts,
} from "../knownHostsDraft";

function makeHost(
  id: number,
  fqdn: string,
  opts: { icon?: string | null; groupIds?: number[] } = {},
): KnownHostWithStats {
  return {
    id,
    fqdn,
    icon: opts.icon ?? null,
    created_at: "2026-01-01T00:00:00Z",
    user_count: 0,
    groups: (opts.groupIds ?? []).map((gid) => ({ id: gid, name: `g${gid}` })),
  };
}

describe("knownHostsDraft reducer", () => {
  it("initialises draft mirroring server state with no diff", () => {
    const state = fromServerHosts([makeHost(1, "a.lan"), makeHost(2, "b.lan")]);
    expect(state.draft.size).toBe(2);
    expect(isDirtyHosts(state)).toBe(false);
    expect(diffHosts(state)).toEqual({
      added: [],
      removed: [],
      iconChanged: [],
      groupsChanged: [],
    });
  });

  it("adds a new draft host with a tagged id", () => {
    const initial = fromServerHosts([]);
    const next = hostsDraftReducer(initial, {
      type: "add",
      id: "new-1",
      host: { fqdn: "new.lan", icon: null, groupIds: [] },
    });
    expect(next.draft.size).toBe(1);
    expect(diffHosts(next).added).toHaveLength(1);
    expect(isDirtyHosts(next)).toBe(true);
  });

  it("removing a new (unsaved) host leaves no trace", () => {
    let state = fromServerHosts([]);
    state = hostsDraftReducer(state, {
      type: "add",
      id: "new-1",
      host: { fqdn: "new.lan", icon: null, groupIds: [] },
    });
    state = hostsDraftReducer(state, { type: "remove", id: "new-1" });

    expect(state.draft.size).toBe(0);
    expect(state.tombstoned.size).toBe(0);
    expect(isDirtyHosts(state)).toBe(false);
  });

  it("removing a persisted host tombstones it", () => {
    let state = fromServerHosts([makeHost(1, "a.lan")]);
    state = hostsDraftReducer(state, { type: "remove", id: 1 });

    expect(state.draft.has(1)).toBe(false);
    expect(state.tombstoned.has(1)).toBe(true);
    expect(diffHosts(state).removed).toHaveLength(1);
  });

  it("restore un-tombstones a removed host", () => {
    let state = fromServerHosts([makeHost(1, "a.lan")]);
    state = hostsDraftReducer(state, { type: "remove", id: 1 });
    state = hostsDraftReducer(state, { type: "restore", id: 1 });

    expect(state.tombstoned.size).toBe(0);
    expect(state.draft.has(1)).toBe(true);
    expect(isDirtyHosts(state)).toBe(false);
  });

  it("tracks icon changes on persisted hosts", () => {
    let state = fromServerHosts([makeHost(1, "a.lan", { icon: "IconServer" })]);
    state = hostsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { icon: "IconCloud" },
    });
    expect(diffHosts(state).iconChanged).toHaveLength(1);
  });

  it("does not flag icon change when the value is unchanged", () => {
    let state = fromServerHosts([makeHost(1, "a.lan", { icon: "IconServer" })]);
    state = hostsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { icon: "IconServer" },
    });
    expect(diffHosts(state).iconChanged).toHaveLength(0);
  });

  it("tracks group-membership changes (order-insensitive)", () => {
    let state = fromServerHosts([
      makeHost(1, "a.lan", { groupIds: [10, 20] }),
    ]);
    state = hostsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { groupIds: [20, 10] },
    });
    expect(diffHosts(state).groupsChanged).toHaveLength(0);

    state = hostsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { groupIds: [10, 30] },
    });
    expect(diffHosts(state).groupsChanged).toHaveLength(1);
  });

  it("discard reverts every change", () => {
    let state = fromServerHosts([makeHost(1, "a.lan", { icon: "IconServer" })]);
    state = hostsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { icon: "IconCloud" },
    });
    state = hostsDraftReducer(state, {
      type: "add",
      id: "new-1",
      host: { fqdn: "new.lan", icon: null, groupIds: [] },
    });
    state = hostsDraftReducer(state, { type: "remove", id: 1 });
    expect(isDirtyHosts(state)).toBe(true);

    state = hostsDraftReducer(state, { type: "discard" });
    expect(isDirtyHosts(state)).toBe(false);
    expect(state.draft.size).toBe(1);
    expect(state.draft.get(1)?.icon).toBe("IconServer");
  });
});
