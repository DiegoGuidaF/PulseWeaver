import { describe, expect, it } from "vitest";
import type { Host } from "@/lib/api";
import {
  diffHosts,
  fromServerHosts,
  hostsDraftReducer,
  isDirtyHosts,
} from "../knownHostsDraft";

function makeHost(
  id: number,
  fqdn: string,
  opts: { groupIds?: number[] } = {},
): Host {
  return {
    id,
    fqdn,
    groups: (opts.groupIds ?? []).map((groupId) => (
      { id: groupId, name: `g${groupId}`, color: "#000000", icon: "server" }
    )),
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
      groupsChanged: [],
    });
  });

  it("adds a new draft host with a tagged id", () => {
    const initial = fromServerHosts([]);
    const next = hostsDraftReducer(initial, {
      type: "add",
      id: "new-1",
      host: { fqdn: "new.lan", groups: [] },
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
      host: { fqdn: "new.lan", groups: [] },
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

  it("tracks group-membership changes", () => {
    let state = fromServerHosts([makeHost(1, "a.lan", { groupIds: [10] })]);
    state = hostsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { groups: [{ id: 10, name: "g10", color: "#000000", icon: "server" }] },
    });
    expect(diffHosts(state).groupsChanged).toHaveLength(0);

    state = hostsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { groups: [{ id: 30, name: "g30", color: "#000000", icon: "server" }] },
    });
    expect(diffHosts(state).groupsChanged).toHaveLength(1);
  });

  it("does not flag a re-ordered but identical group set as changed", () => {
    let state = fromServerHosts([makeHost(1, "a.lan", { groupIds: [10, 20] })]);
    state = hostsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: {
        groups: [
          { id: 20, name: "g20", color: "#000000", icon: "server" },
          { id: 10, name: "g10", color: "#000000", icon: "server" },
        ],
      },
    });
    expect(diffHosts(state).groupsChanged).toHaveLength(0);
  });

  it("discard reverts every change, including a host with multiple groups", () => {
    let state = fromServerHosts([makeHost(1, "a.lan", { groupIds: [10, 20] })]);
    state = hostsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { groups: [{ id: 20, name: "g20", color: "#000000", icon: "server" }] },
    });
    state = hostsDraftReducer(state, {
      type: "add",
      id: "new-1",
      host: { fqdn: "new.lan", groups: [] },
    });
    state = hostsDraftReducer(state, { type: "remove", id: 1 });
    expect(isDirtyHosts(state)).toBe(true);

    state = hostsDraftReducer(state, { type: "discard" });
    expect(isDirtyHosts(state)).toBe(false);
    expect(state.draft.size).toBe(1);
    expect(state.draft.get(1)?.groups.map((g) => g.id)).toEqual([10, 20]);
  });
});
