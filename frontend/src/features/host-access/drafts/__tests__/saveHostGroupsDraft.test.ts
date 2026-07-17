import { describe, expect, it } from "vitest";
import type { GroupDetailWithUsers, GroupListItem } from "@/lib/api";
import { fromServerGroups, groupsDraftReducer, type GroupsDraftState } from "../hostGroupsDraft";
import {
  buildReconcileGroupsBody,
  groupsOriginalMatchesServer,
  unvisitedExistingGroupIds,
} from "../saveHostGroupsDraft";

function makeGroup(
  id: number,
  name: string,
  opts: {
    hostIds?: number[];
    icon?: string;
    description?: string | null;
    color?: string;
  } = {},
): GroupDetailWithUsers {
  return {
    id,
    name,
    description: opts.description ?? null,
    icon: opts.icon ?? "server",
    color: opts.color ?? "#000000",
    hosts: (opts.hostIds ?? []).map((hid) => ({ id: hid, fqdn: `h${hid}.lan` })),
    network_policies: [],
    users: [],
  };
}

function toListItem(g: GroupDetailWithUsers): GroupListItem {
  return { id: g.id, name: g.name, color: g.color, icon: g.icon, host_count: g.hosts.length };
}

/** Seeds a state with every group visited (detail hydrated) — see hostGroupsDraft.test.ts. */
function seedVisited(groups: GroupDetailWithUsers[] = []): GroupsDraftState {
  let state: GroupsDraftState = { ...fromServerGroups(groups.map(toListItem)), selectedId: null };
  for (const g of groups) {
    state = groupsDraftReducer(state, { type: "hydrateDetail", detail: g });
  }
  return state;
}

/** Seeds a state from the light list only — no group has been visited. */
function seedUnvisited(groups: GroupDetailWithUsers[] = []): GroupsDraftState {
  return { ...fromServerGroups(groups.map(toListItem)), selectedId: null };
}

describe("buildReconcileGroupsBody", () => {
  it("projects a persisted, visited group with its numeric id and all optional fields", () => {
    const state = seedVisited([
      makeGroup(1, "infra", {
        hostIds: [10, 20],
        icon: "🏗️",
        description: "infra hosts",
        color: "#336699",
      }),
    ]);

    const body = buildReconcileGroupsBody(state, new Map());

    expect(body).toEqual([
      {
        id: 1,
        name: "infra",
        description: "infra hosts",
        icon: "🏗️",
        color: "#336699",
        host_ids: [10, 20],
      },
    ]);
  });

  it("falls back to freshDetails for an unvisited existing group", () => {
    const state = seedUnvisited([makeGroup(1, "infra")]);
    const fresh = new Map([[1, makeGroup(1, "infra", { hostIds: [7, 8] })]]);

    const body = buildReconcileGroupsBody(state, fresh);

    expect(body[0]?.host_ids).toEqual([7, 8]);
  });

  it("preserves color picked on a new draft group", () => {
    const state = groupsDraftReducer(seedVisited([]), {
      type: "add",
      id: "new-color",
      group: {
        name: "tagged",
        description: null,
        icon: null,
        color: "#7950F2",
        hostIds: [],
      },
    });

    const body = buildReconcileGroupsBody(state, new Map());

    expect(body[0]?.color).toBe("#7950F2");
  });

  it("projects a new draft group with id: null, uses empty string fallback for icon", () => {
    const state = groupsDraftReducer(seedVisited([]), {
      type: "add",
      id: "new-zzz",
      group: {
        name: "fresh",
        description: null,
        icon: null,
        color: "#4C6EF5",
        hostIds: [],
      },
    });

    const body = buildReconcileGroupsBody(state, new Map());

    expect(body).toEqual([
      { id: null, name: "fresh", description: null, icon: "", color: "#4C6EF5", host_ids: [] },
    ]);
  });

  it("omits tombstoned groups (remove drops them from draft)", () => {
    const initial = seedVisited([makeGroup(1, "keep"), makeGroup(2, "drop")]);
    const state = groupsDraftReducer(initial, { type: "remove", id: 2 });

    const body = buildReconcileGroupsBody(state, new Map());

    expect(body.map((g) => g.name)).toEqual(["keep"]);
    expect(state.tombstoned.has(2)).toBe(true);
  });
});

describe("unvisitedExistingGroupIds", () => {
  it("returns ids for existing groups that haven't been visited", () => {
    const state = seedUnvisited([makeGroup(1, "a"), makeGroup(2, "b")]);
    expect(unvisitedExistingGroupIds(state).sort()).toEqual([1, 2]);
  });

  it("excludes visited groups and new (unsaved) groups", () => {
    let state = seedVisited([makeGroup(1, "a")]);
    state = groupsDraftReducer(state, {
      type: "add",
      id: "new-1",
      group: { name: "b", description: null, icon: null, color: "#4C6EF5", hostIds: [] },
    });
    expect(unvisitedExistingGroupIds(state)).toEqual([]);
  });
});

describe("groupsOriginalMatchesServer", () => {
  it("returns true when current ids match the original set exactly", () => {
    const original = seedUnvisited([makeGroup(1, "a"), makeGroup(2, "b")]).listOriginal;

    expect(
      groupsOriginalMatchesServer(original, [toListItem(makeGroup(1, "a")), toListItem(makeGroup(2, "b"))]),
    ).toBe(true);
  });

  it("returns false when sizes differ", () => {
    const original = seedUnvisited([makeGroup(1, "a")]).listOriginal;

    expect(
      groupsOriginalMatchesServer(original, [toListItem(makeGroup(1, "a")), toListItem(makeGroup(2, "b"))]),
    ).toBe(false);
  });

  it("returns false when an id is missing even with matching size", () => {
    const original = seedUnvisited([makeGroup(1, "a"), makeGroup(2, "b")]).listOriginal;

    expect(
      groupsOriginalMatchesServer(original, [toListItem(makeGroup(1, "a")), toListItem(makeGroup(3, "c"))]),
    ).toBe(false);
  });
});
