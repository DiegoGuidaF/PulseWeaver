import { describe, expect, it } from "vitest";
import type { HostGroupWithMembers } from "@/lib/api";
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
    icon?: string | null;
    description?: string | null;
    color?: string | null;
  } = {},
): HostGroupWithMembers {
  return {
    id,
    name,
    description: opts.description ?? null,
    icon: opts.icon ?? null,
    color: opts.color ?? null,
    created_at: "2026-01-01T00:00:00Z",
    hosts: (opts.hostIds ?? []).map((hid) => ({ id: hid, fqdn: `h${hid}.lan` })),
    member_ids: opts.hostIds ?? [],
  };
}

function seed(groups: HostGroupWithMembers[] = []) {
  // Mirrors hostGroupsDraft.test.ts: fromServerGroups returns Omit<..., "selectedId">.
  return { ...fromServerGroups(groups), selectedId: null };
}

describe("buildReconcileGroupsBody", () => {
  it("projects a persisted group with its numeric id and all optional fields", () => {
    // NOTE: server-side color is currently dropped by fromServerGroups (see the
    // "persisted once backend adds the column" comment there). Once the column
    // lands and fromServerGroups starts hydrating it, this test will flip to
    // expect "indigo" — and that flip is exactly the contract guard we want.
    const state = seed([
      makeGroup(1, "infra", {
        hostIds: [10, 20],
        icon: "🏗️",
        description: "infra hosts",
        color: "indigo",
      }),
    ]);

    const body = buildReconcileGroupsBody(state);

    expect(body).toEqual([
      {
        id: 1,
        name: "infra",
        description: "infra hosts",
        icon: "🏗️",
        color: null,
        host_ids: [10, 20],
      },
    ]);
  });

  it("preserves color picked on a new draft group (the only path color flows through today)", () => {
    const state = groupsDraftReducer(seed([]), {
      type: "add",
      id: "new-color",
      group: {
        name: "tagged",
        description: null,
        icon: null,
        color: "violet",
        hostIds: [],
      },
    });

    const body = buildReconcileGroupsBody(state);

    expect(body[0]?.color).toBe("violet");
  });

  it("projects a new draft group with id: null", () => {
    const state = groupsDraftReducer(seed([]), {
      type: "add",
      id: "new-zzz",
      group: {
        name: "fresh",
        description: null,
        icon: null,
        color: null,
        hostIds: [],
      },
    });

    const body = buildReconcileGroupsBody(state);

    expect(body).toEqual([
      { id: null, name: "fresh", description: null, icon: null, color: null, host_ids: [] },
    ]);
  });

  it("preserves null optional fields verbatim", () => {
    const state = seed([makeGroup(1, "g", { hostIds: [] })]);

    const body = buildReconcileGroupsBody(state);

    expect(body[0]).toMatchObject({ description: null, icon: null, color: null });
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
