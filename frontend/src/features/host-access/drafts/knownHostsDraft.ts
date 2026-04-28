import type { Id, KnownHostWithStats } from "@/lib/api";

export type DraftHostId = Id | `new-${string}`;

export interface DraftHost {
  id: DraftHostId;
  fqdn: string;
  icon: string | null;
  groupIds: Id[];
  source?: "suggestion";
}

export interface HostsDraftState {
  original: Map<Id, KnownHostWithStats>;
  draft: Map<DraftHostId, DraftHost>;
  tombstoned: Set<Id>;
}

export type HostsDraftAction =
  | { type: "reset"; hosts: KnownHostWithStats[] }
  | { type: "add"; id: `new-${string}`; host: Omit<DraftHost, "id"> }
  | { type: "update"; id: DraftHostId; patch: Partial<Omit<DraftHost, "id">> }
  | { type: "remove"; id: DraftHostId }
  | { type: "restore"; id: Id }
  | { type: "discard" };

export function initialHostsDraft(): HostsDraftState {
  return { original: new Map(), draft: new Map(), tombstoned: new Set() };
}

export function fromServerHosts(hosts: KnownHostWithStats[]): HostsDraftState {
  const original = new Map<Id, KnownHostWithStats>();
  const draft = new Map<DraftHostId, DraftHost>();
  for (const h of hosts) {
    original.set(h.id, h);
    draft.set(h.id, {
      id: h.id,
      fqdn: h.fqdn,
      icon: h.icon ?? null,
      groupIds: h.groups.map((g) => g.id),
    });
  }
  return { original, draft, tombstoned: new Set() };
}

export function hostsDraftReducer(
  state: HostsDraftState,
  action: HostsDraftAction,
): HostsDraftState {
  switch (action.type) {
    case "reset":
      return fromServerHosts(action.hosts);

    case "add": {
      const draft = new Map(state.draft);
      draft.set(action.id, { id: action.id, ...action.host });
      return { ...state, draft };
    }

    case "update": {
      const existing = state.draft.get(action.id);
      if (!existing) return state;
      const draft = new Map(state.draft);
      draft.set(action.id, { ...existing, ...action.patch, id: existing.id });
      return { ...state, draft };
    }

    case "remove": {
      const draft = new Map(state.draft);
      draft.delete(action.id);
      // New drafts leave no trace; persisted hosts get tombstoned.
      if (typeof action.id === "number") {
        const tombstoned = new Set(state.tombstoned);
        tombstoned.add(action.id);
        return { ...state, draft, tombstoned };
      }
      return { ...state, draft };
    }

    case "restore": {
      if (!state.tombstoned.has(action.id)) return state;
      const tombstoned = new Set(state.tombstoned);
      tombstoned.delete(action.id);
      const originalHost = state.original.get(action.id);
      const draft = new Map(state.draft);
      if (originalHost) {
        draft.set(action.id, {
          id: action.id,
          fqdn: originalHost.fqdn,
          icon: originalHost.icon ?? null,
          groupIds: originalHost.groups.map((g) => g.id),
        });
      }
      return { ...state, draft, tombstoned };
    }

    case "discard":
      return fromServerHosts(Array.from(state.original.values()));
  }
}

export interface HostsDiff {
  added: DraftHost[];
  removed: KnownHostWithStats[];
  iconChanged: DraftHost[];
  groupsChanged: DraftHost[];
}

export function diffHosts(state: HostsDraftState): HostsDiff {
  const added: DraftHost[] = [];
  const iconChanged: DraftHost[] = [];
  const groupsChanged: DraftHost[] = [];

  for (const entry of state.draft.values()) {
    if (typeof entry.id !== "number") {
      added.push(entry);
      continue;
    }
    const original = state.original.get(entry.id);
    if (!original) continue;
    if ((original.icon ?? null) !== entry.icon) iconChanged.push(entry);
    if (!sameIds(original.groups.map((g) => g.id), entry.groupIds)) {
      groupsChanged.push(entry);
    }
  }

  const removed: KnownHostWithStats[] = [];
  for (const id of state.tombstoned) {
    const original = state.original.get(id);
    if (original) removed.push(original);
  }

  return { added, removed, iconChanged, groupsChanged };
}

export function isDirtyHosts(state: HostsDraftState): boolean {
  const d = diffHosts(state);
  return (
    d.added.length > 0 ||
    d.removed.length > 0 ||
    d.iconChanged.length > 0 ||
    d.groupsChanged.length > 0
  );
}

export function summarizeHosts(diff: HostsDiff): string {
  const parts: string[] = [];
  if (diff.added.length) parts.push(`${diff.added.length} added`);
  if (diff.removed.length) parts.push(`${diff.removed.length} removed`);
  if (diff.iconChanged.length) parts.push(`${diff.iconChanged.length} icon`);
  if (diff.groupsChanged.length) parts.push(`${diff.groupsChanged.length} re-grouped`);
  return parts.length === 0 ? "No staged changes" : parts.join(" · ");
}

export function hostUserImpact(diff: HostsDiff): string | null {
  const total = diff.removed.reduce((acc, h) => acc + h.user_count, 0);
  if (total === 0) return null;
  return `${total} user${total === 1 ? "" : "s"} will lose access on save`;
}

function sameIds(a: Id[], b: Id[]): boolean {
  if (a.length !== b.length) return false;
  const sortedA = [...a].sort((x, y) => x - y);
  const sortedB = [...b].sort((x, y) => x - y);
  for (let i = 0; i < sortedA.length; i++) {
    if (sortedA[i] !== sortedB[i]) return false;
  }
  return true;
}
