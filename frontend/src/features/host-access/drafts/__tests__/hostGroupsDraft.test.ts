import { describe, expect, it } from "vitest";
import type { GroupDetailWithUsers } from "@/lib/api";
import {
  diffGroups,
  fromServerGroups,
  groupsDraftReducer,
  isDirtyGroups,
} from "../hostGroupsDraft";

function makeGroup(
  id: number,
  name: string,
  opts: { hostIds?: number[]; icon?: string; color?: string; description?: string | null } = {},
): GroupDetailWithUsers {
  return {
    id,
    name,
    description: opts.description ?? null,
    icon: opts.icon ?? "server",
    color: opts.color ?? "#000000",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    hosts: (opts.hostIds ?? []).map((hid) => ({ id: hid, fqdn: `h${hid}.lan` })),
    network_policies: [],
    users: [],
  };
}

function seed(groups: GroupDetailWithUsers[] = []) {
  const server = fromServerGroups(groups);
  const firstId = server.draft.keys().next().value ?? null;
  return { ...server, selectedId: firstId };
}

describe("hostGroupsDraft reducer", () => {
  it("initialises selectedId to the first group", () => {
    const state = seed([makeGroup(1, "a"), makeGroup(2, "b")]);
    expect(state.selectedId).toBe(1);
    expect(isDirtyGroups(state)).toBe(false);
  });

  it("hydrates icon and color from server data", () => {
    const state = seed([makeGroup(1, "a", { icon: "database", color: "#ff0000" })]);
    expect(state.draft.get(1)?.icon).toBe("database");
    expect(state.draft.get(1)?.color).toBe("#ff0000");
  });

  it("adding a new group selects it", () => {
    let state = seed([]);
    state = groupsDraftReducer(state, {
      type: "add",
      id: "new-1",
      group: {
        name: "media",
        description: null,
        icon: null,
        color: null,
        hostIds: [],
      },
    });
    expect(state.selectedId).toBe("new-1");
    expect(diffGroups(state).added).toHaveLength(1);
  });

  it("removing an unsaved group leaves no tombstone", () => {
    let state = seed([]);
    state = groupsDraftReducer(state, {
      type: "add",
      id: "new-1",
      group: {
        name: "media",
        description: null,
        icon: null,
        color: null,
        hostIds: [],
      },
    });
    state = groupsDraftReducer(state, { type: "remove", id: "new-1" });
    expect(state.tombstoned.size).toBe(0);
    expect(isDirtyGroups(state)).toBe(false);
  });

  it("removing a persisted group tombstones and reselects", () => {
    let state = seed([makeGroup(1, "a"), makeGroup(2, "b")]);
    expect(state.selectedId).toBe(1);
    state = groupsDraftReducer(state, { type: "remove", id: 1 });
    expect(state.tombstoned.has(1)).toBe(true);
    expect(state.selectedId).toBe(2);
    expect(diffGroups(state).removed).toHaveLength(1);
  });

  it("toggleHost adds then removes the host id", () => {
    let state = seed([makeGroup(1, "a", { hostIds: [] })]);
    state = groupsDraftReducer(state, { type: "toggleHost", id: 1, hostId: 99 });
    expect(state.draft.get(1)?.hostIds).toEqual([99]);
    expect(diffGroups(state).changed[0].hostsAdded).toEqual([99]);

    state = groupsDraftReducer(state, { type: "toggleHost", id: 1, hostId: 99 });
    expect(state.draft.get(1)?.hostIds).toEqual([]);
    expect(isDirtyGroups(state)).toBe(false);
  });

  it("tracks metadata changes individually", () => {
    let state = seed([makeGroup(1, "a", { icon: "database", description: "x" })]);
    state = groupsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { name: "a-renamed" },
    });
    const diff = diffGroups(state);
    expect(diff.changed[0].nameChanged).toBe(true);
    expect(diff.changed[0].iconChanged).toBe(false);
    expect(diff.changed[0].descriptionChanged).toBe(false);
  });

  it("exposes per-id diff in byId map", () => {
    let state = seed([makeGroup(1, "a"), makeGroup(2, "b")]);
    state = groupsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { name: "renamed" },
    });
    state = groupsDraftReducer(state, { type: "remove", id: 2 });
    state = groupsDraftReducer(state, {
      type: "add",
      id: "new-1",
      group: { name: "c", description: null, icon: null, color: null, hostIds: [] },
    });

    const byId = diffGroups(state).byId;
    expect(byId.get(2)).toBe("removed");
    expect(byId.get("new-1")).toBe("added");
    const first = byId.get(1);
    expect(typeof first).toBe("object");
    if (typeof first === "object" && first !== null) {
      expect(first.nameChanged).toBe(true);
    }
  });

  it("discard reverts all state but keeps the selected id pointing at something sensible", () => {
    let state = seed([makeGroup(1, "a"), makeGroup(2, "b")]);
    state = groupsDraftReducer(state, {
      type: "update",
      id: 1,
      patch: { name: "renamed" },
    });
    state = groupsDraftReducer(state, { type: "discard" });
    expect(isDirtyGroups(state)).toBe(false);
    expect(state.draft.get(1)?.name).toBe("a");
  });
});
