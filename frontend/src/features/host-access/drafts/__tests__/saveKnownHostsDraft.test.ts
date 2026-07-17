import { describe, expect, it } from "vitest";
import type { Host } from "@/lib/api";
import { fromServerHosts, hostsDraftReducer } from "../knownHostsDraft";
import {
  buildReconcileHostsBody,
  hostsOriginalMatchesServer,
} from "../saveKnownHostsDraft";

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

describe("buildReconcileHostsBody", () => {
  it("projects a persisted host with its numeric id", () => {
    const state = fromServerHosts([makeHost(1, "a.lan", { groupIds: [10] })]);

    const body = buildReconcileHostsBody(state);

    expect(body).toEqual([{ id: 1, fqdn: "a.lan", group_ids: [10] }]);
  });

  it("projects a new draft host with id: null", () => {
    const state = hostsDraftReducer(fromServerHosts([]), {
      type: "add",
      id: "new-abc",
      host: { fqdn: "fresh.lan", groups: [] },
    });

    const body = buildReconcileHostsBody(state);

    expect(body).toEqual([{ id: null, fqdn: "fresh.lan", group_ids: [] }]);
  });

  it("omits tombstoned hosts (remove drops them from draft)", () => {
    const initial = fromServerHosts([makeHost(1, "keep.lan"), makeHost(2, "drop.lan")]);
    const state = hostsDraftReducer(initial, { type: "remove", id: 2 });

    const body = buildReconcileHostsBody(state);

    expect(body).toHaveLength(1);
    expect(body[0]?.fqdn).toBe("keep.lan");
    expect(state.tombstoned.has(2)).toBe(true);
  });

  it("sends an empty group_ids array for an unassigned host", () => {
    const state = fromServerHosts([makeHost(1, "a.lan")]);

    const body = buildReconcileHostsBody(state);

    expect(body[0]).toEqual({ id: 1, fqdn: "a.lan", group_ids: [] });
  });

  it("emits one entry per draft entry, regardless of mix of persisted and new", () => {
    let state = fromServerHosts([makeHost(1, "old.lan", { groupIds: [10] })]);
    state = hostsDraftReducer(state, {
      type: "add",
      id: "new-xyz",
      host: { fqdn: "new.lan", groups: [{ id: 20, name: "g20", color: "#000000", icon: "server" }] },
    });

    const body = buildReconcileHostsBody(state);

    expect(body).toHaveLength(2);
    expect(body.find((h) => h.fqdn === "old.lan")).toEqual({
      id: 1,
      fqdn: "old.lan",
      group_ids: [10],
    });
    expect(body.find((h) => h.fqdn === "new.lan")).toEqual({
      id: null,
      fqdn: "new.lan",
      group_ids: [20],
    });
  });

  it("sends every membership for a host in multiple groups", () => {
    const state = fromServerHosts([makeHost(1, "multi.lan", { groupIds: [10, 20] })]);

    const body = buildReconcileHostsBody(state);

    expect(body).toEqual([{ id: 1, fqdn: "multi.lan", group_ids: [10, 20] }]);
  });

  it("preserves an untouched host's multiple groups when another host in the same draft is edited", () => {
    let state = fromServerHosts([
      makeHost(1, "multi.lan", { groupIds: [10, 20] }),
      makeHost(2, "other.lan", { groupIds: [10] }),
    ]);
    state = hostsDraftReducer(state, {
      type: "update",
      id: 2,
      patch: { groups: [{ id: 30, name: "g30", color: "#000000", icon: "server" }] },
    });

    const body = buildReconcileHostsBody(state);

    expect(body.find((h) => h.fqdn === "multi.lan")).toEqual({
      id: 1,
      fqdn: "multi.lan",
      group_ids: [10, 20],
    });
    expect(body.find((h) => h.fqdn === "other.lan")).toEqual({
      id: 2,
      fqdn: "other.lan",
      group_ids: [30],
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

  it("returns false when a host's group memberships changed with the same id set", () => {
    const original = fromServerHosts([makeHost(1, "a.lan", { groupIds: [10, 20] })]).original;

    expect(
      hostsOriginalMatchesServer(original, [makeHost(1, "a.lan", { groupIds: [10] })]),
    ).toBe(false);
  });

  it("returns true when group memberships match regardless of order", () => {
    const original = fromServerHosts([makeHost(1, "a.lan", { groupIds: [10, 20] })]).original;

    expect(
      hostsOriginalMatchesServer(original, [makeHost(1, "a.lan", { groupIds: [20, 10] })]),
    ).toBe(true);
  });
});
