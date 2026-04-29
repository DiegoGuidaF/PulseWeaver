import type { Id, HostGroupWithMembers } from "@/lib/api";

export type DraftGroupId = Id | `new-${string}`;

export const GROUP_COLOR_PALETTE = [
  "indigo",
  "violet",
  "teal",
  "cyan",
  "grape",
  "pink",
  "lime",
  "green",
  "gray",
] as const;

export type GroupColor = (typeof GROUP_COLOR_PALETTE)[number];

export interface DraftGroup {
  id: DraftGroupId;
  name: string;
  description: string | null;
  icon: string | null;
  color: GroupColor | null;
  hostIds: Id[];
}

export interface GroupsDraftState {
  original: Map<Id, HostGroupWithMembers>;
  draft: Map<DraftGroupId, DraftGroup>;
  tombstoned: Set<Id>;
  selectedId: DraftGroupId | null;
}

export type GroupsDraftAction =
  | { type: "reset"; groups: HostGroupWithMembers[] }
  | { type: "add"; id: `new-${string}`; group: Omit<DraftGroup, "id"> }
  | { type: "update"; id: DraftGroupId; patch: Partial<Omit<DraftGroup, "id">> }
  | { type: "remove"; id: DraftGroupId }
  | { type: "restore"; id: Id }
  | { type: "select"; id: DraftGroupId | null }
  | { type: "toggleHost"; id: DraftGroupId; hostId: Id }
  | { type: "discard" };

export function initialGroupsDraft(): GroupsDraftState {
  return {
    original: new Map(),
    draft: new Map(),
    tombstoned: new Set(),
    selectedId: null,
  };
}

export function fromServerGroups(
  groups: HostGroupWithMembers[],
): Omit<GroupsDraftState, "selectedId"> {
  const original = new Map<Id, HostGroupWithMembers>();
  const draft = new Map<DraftGroupId, DraftGroup>();
  for (const g of groups) {
    original.set(g.id, g);
    draft.set(g.id, {
      id: g.id,
      name: g.name,
      description: g.description ?? null,
      icon: g.icon ?? null,
      color: null, // persisted once backend adds the column
      hostIds: g.hosts.map((h) => h.id),
    });
  }
  return { original, draft, tombstoned: new Set() };
}

export function groupsDraftReducer(
  state: GroupsDraftState,
  action: GroupsDraftAction,
): GroupsDraftState {
  switch (action.type) {
    case "reset": {
      const next = fromServerGroups(action.groups);
      const firstId = next.draft.keys().next().value ?? null;
      return { ...next, selectedId: firstId };
    }

    case "add": {
      const draft = new Map(state.draft);
      draft.set(action.id, { id: action.id, ...action.group });
      return { ...state, draft, selectedId: action.id };
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
      const tombstoned =
        typeof action.id === "number"
          ? new Set(state.tombstoned).add(action.id)
          : state.tombstoned;
      // Move selection to the first remaining draft entry when the selected group is removed.
      const nextSelected =
        state.selectedId === action.id
          ? (draft.keys().next().value ?? null)
          : state.selectedId;
      return { ...state, draft, tombstoned, selectedId: nextSelected };
    }

    case "restore": {
      if (!state.tombstoned.has(action.id)) return state;
      const tombstoned = new Set(state.tombstoned);
      tombstoned.delete(action.id);
      const original = state.original.get(action.id);
      const draft = new Map(state.draft);
      if (original) {
        draft.set(action.id, {
          id: action.id,
          name: original.name,
          description: original.description ?? null,
          icon: original.icon ?? null,
          color: null,
          hostIds: original.hosts.map((h) => h.id),
        });
      }
      return { ...state, draft, tombstoned };
    }

    case "select":
      return { ...state, selectedId: action.id };

    case "toggleHost": {
      const existing = state.draft.get(action.id);
      if (!existing) return state;
      const has = existing.hostIds.includes(action.hostId);
      const hostIds = has
        ? existing.hostIds.filter((h) => h !== action.hostId)
        : [...existing.hostIds, action.hostId];
      const draft = new Map(state.draft);
      draft.set(action.id, { ...existing, hostIds });
      return { ...state, draft };
    }

    case "discard": {
      const next = fromServerGroups(Array.from(state.original.values()));
      return { ...next, selectedId: state.selectedId };
    }
  }
}

export interface GroupDiffEntry {
  group: DraftGroup;
  nameChanged: boolean;
  descriptionChanged: boolean;
  iconChanged: boolean;
  colorChanged: boolean;
  hostsAdded: Id[];
  hostsRemoved: Id[];
}

export interface GroupsDiff {
  added: DraftGroup[];
  removed: HostGroupWithMembers[];
  changed: GroupDiffEntry[];
  byId: Map<DraftGroupId, GroupDiffEntry | "added" | "removed">;
}

export function diffGroups(state: GroupsDraftState): GroupsDiff {
  const added: DraftGroup[] = [];
  const changed: GroupDiffEntry[] = [];
  const byId = new Map<DraftGroupId, GroupDiffEntry | "added" | "removed">();

  for (const entry of state.draft.values()) {
    if (typeof entry.id !== "number") {
      added.push(entry);
      byId.set(entry.id, "added");
      continue;
    }
    const original = state.original.get(entry.id);
    if (!original) continue;
    const diffEntry = computeGroupDiff(entry, original);
    if (isGroupEntryDirty(diffEntry)) {
      changed.push(diffEntry);
      byId.set(entry.id, diffEntry);
    }
  }

  const removed: HostGroupWithMembers[] = [];
  for (const id of state.tombstoned) {
    const original = state.original.get(id);
    if (original) {
      removed.push(original);
      byId.set(id, "removed");
    }
  }

  return { added, removed, changed, byId };
}

export function isDirtyGroups(state: GroupsDraftState): boolean {
  const d = diffGroups(state);
  return d.added.length > 0 || d.removed.length > 0 || d.changed.length > 0;
}

function computeGroupDiff(
  draft: DraftGroup,
  original: HostGroupWithMembers,
): GroupDiffEntry {
  const originalHostIds = new Set(original.hosts.map((h) => h.id));
  const draftHostIds = new Set(draft.hostIds);
  const hostsAdded = draft.hostIds.filter((id) => !originalHostIds.has(id));
  const hostsRemoved = original.hosts
    .map((h) => h.id)
    .filter((id) => !draftHostIds.has(id));

  return {
    group: draft,
    nameChanged: draft.name !== original.name,
    descriptionChanged: (draft.description ?? null) !== (original.description ?? null),
    iconChanged: (draft.icon ?? null) !== (original.icon ?? null),
    colorChanged: draft.color !== null,
    hostsAdded,
    hostsRemoved,
  };
}

export function toDraftFromOriginal(state: GroupsDraftState, id: Id): DraftGroup | null {
  const original = state.original.get(id);
  if (!original) return null;
  return {
    id,
    name: original.name,
    description: original.description ?? null,
    icon: original.icon ?? null,
    color: null,
    hostIds: original.hosts.map((h) => h.id),
  };
}

export function summarizeGroups(diff: GroupsDiff): string {
  const parts: string[] = [];
  if (diff.added.length) parts.push(`${diff.added.length} added`);
  if (diff.removed.length) parts.push(`${diff.removed.length} removed`);
  if (diff.changed.length) parts.push(`${diff.changed.length} changed`);
  return parts.length === 0 ? "No staged changes" : parts.join(" · ");
}

function isGroupEntryDirty(e: GroupDiffEntry): boolean {
  return (
    e.nameChanged ||
    e.descriptionChanged ||
    e.iconChanged ||
    e.colorChanged ||
    e.hostsAdded.length > 0 ||
    e.hostsRemoved.length > 0
  );
}

