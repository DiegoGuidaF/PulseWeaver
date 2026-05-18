import { describe, expect, it } from "vitest";
import type { GroupDetailWithUsers } from "@/lib/api";
import { fromServerGroups, groupsDraftReducer } from "../hostGroupsDraft";
import {
  buildReconcileGroupsBody,
  groupsOriginalMatchesServer,
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
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    hosts: (opts.hostIds ?? []).map((hid) => ({ id: hid, fqdn: `h${hid}.lan` })),
    network_policies: [],
    users: [],
  };
}

function seed(groups: GroupDetailWithUsers[] = []) {
  return { ...fromServerGroups(groups), selectedId: null };
}

describe("buildReconcileGroupsBody", () => {
  it("projects a persisted group with its numeric id and all optional fields", () => {
    const state = seed([
      makeGroup(1, "infra", {
        hostIds: [10, 20],
        icon: "🏗️",
        description: "infra hosts",
        color: "#336699",
      }),
    ]);

    const body = buildReconcileGroupsBody(state);

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

  it("preserves color picked on a new draft group", () => {
    const state = groupsDraftReducer(seed([]), {
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

    const body = buildReconcileGroupsBody(state);

    expect(body[0]?.color).toBe("#7950F2");
  });

  it("projects a new draft group with id: null, uses empty string fallback for icon", () => {
    const state = groupsDraftReducer(seed([]), {
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

    const body = buildReconcileGroupsBody(state);

    expect(body).toEqual([
      { id: null, name: "fresh", description: null, icon: "", color: "#4C6EF5", host_ids: [] },
    ]);
  });

  it("omits tombstoned groups (remove drops them from draft)", () => {
    const initial = seed([makeGroup(1, "keep"), makeGroup(2, "drop")]);
    const state = groupsDraftReducer(initial, { type: "remove", id: 2 });

    const body = buildReconcileGroupsBody(state);

    expect(body.map((g) => g.name)).toEqual(["keep"]);
    expect(state.tombstoned.has(2)).toBe(true);
  });
});

describe("groupsOriginalMatchesServer", () => {
  it("returns true when current ids match the original set exactly", () => {
    const original = seed([makeGroup(1, "a"), makeGroup(2, "b")]).original;

    expect(groupsOriginalMatchesServer(original, [makeGroup(1, "a"), makeGroup(2, "b")])).toBe(
      true,
    );
  });

  it("returns false when sizes differ", () => {
    const original = seed([makeGroup(1, "a")]).original;

    expect(groupsOriginalMatchesServer(original, [makeGroup(1, "a"), makeGroup(2, "b")])).toBe(
      false,
    );
  });

  it("returns false when an id is missing even with matching size", () => {
    const original = seed([makeGroup(1, "a"), makeGroup(2, "b")]).original;

    expect(groupsOriginalMatchesServer(original, [makeGroup(1, "a"), makeGroup(3, "c")])).toBe(
      false,
    );
  });
});
