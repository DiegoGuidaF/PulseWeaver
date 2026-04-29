import { describe, expect, it } from "vitest";
import type { KnownHostWithStats } from "@/lib/api";
import { fromServerHosts, hostsDraftReducer } from "../knownHostsDraft";
import {
  buildReconcileHostsBody,
  hostsOriginalMatchesServer,
} from "../saveKnownHostsDraft";

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

describe("buildReconcileHostsBody", () => {
  it("projects a persisted host with its numeric id", () => {
    const state = fromServerHosts([makeHost(1, "a.lan", { groupIds: [10] })]);

    const body = buildReconcileHostsBody(state);

    expect(body).toEqual([{ id: 1, fqdn: "a.lan", icon: null, group_ids: [10] }]);
  });

  it("projects a new draft host with id: null", () => {
    const state = hostsDraftReducer(fromServerHosts([]), {
      type: "add",
      id: "new-abc",
      host: { fqdn: "fresh.lan", icon: "🆕", groupIds: [] },
    });

    const body = buildReconcileHostsBody(state);

    expect(body).toEqual([{ id: null, fqdn: "fresh.lan", icon: "🆕", group_ids: [] }]);
  });

  it("omits tombstoned hosts (remove drops them from draft)", () => {
    const initial = fromServerHosts([makeHost(1, "keep.lan"), makeHost(2, "drop.lan")]);
    const state = hostsDraftReducer(initial, { type: "remove", id: 2 });

    const body = buildReconcileHostsBody(state);

    expect(body).toHaveLength(1);
    expect(body[0]?.fqdn).toBe("keep.lan");
    // Sanity: tombstone tracking still happened, even though it's not in the body.
    expect(state.tombstoned.has(2)).toBe(true);
  });

  it("preserves null icon and groupIds order verbatim", () => {
    const state = fromServerHosts([makeHost(1, "a.lan", { icon: null, groupIds: [3, 1, 2] })]);

    const body = buildReconcileHostsBody(state);

    expect(body[0]).toEqual({ id: 1, fqdn: "a.lan", icon: null, group_ids: [3, 1, 2] });
  });

  it("emits one entry per draft entry, regardless of mix of persisted and new", () => {
    let state = fromServerHosts([makeHost(1, "old.lan", { groupIds: [10] })]);
    state = hostsDraftReducer(state, {
      type: "add",
      id: "new-xyz",
      host: { fqdn: "new.lan", icon: null, groupIds: [10, 20] },
    });

    const body = buildReconcileHostsBody(state);

    expect(body).toHaveLength(2);
    expect(body.find((h) => h.fqdn === "old.lan")).toEqual({
      id: 1,
      fqdn: "old.lan",
      icon: null,
      group_ids: [10],
    });
    expect(body.find((h) => h.fqdn === "new.lan")).toEqual({
      id: null,
      fqdn: "new.lan",
      icon: null,
      group_ids: [10, 20],
    });
  });
});

describe("hostsOriginalMatchesServer", () => {
  it("returns true when current ids match the original set exactly", () => {
    const original = fromServerHosts([makeHost(1, "a.lan"), makeHost(2, "b.lan")]).original;

    expect(hostsOriginalMatchesServer(original, [makeHost(1, "a.lan"), makeHost(2, "b.lan")])).toBe(
      true,
    );
  });

  it("returns false when sizes differ", () => {
    const original = fromServerHosts([makeHost(1, "a.lan")]).original;

    expect(hostsOriginalMatchesServer(original, [makeHost(1, "a.lan"), makeHost(2, "b.lan")])).toBe(
      false,
    );
  });

  it("returns false when an id is missing even with matching size", () => {
    const original = fromServerHosts([makeHost(1, "a.lan"), makeHost(2, "b.lan")]).original;

    expect(hostsOriginalMatchesServer(original, [makeHost(1, "a.lan"), makeHost(3, "c.lan")])).toBe(
      false,
    );
  });
});
