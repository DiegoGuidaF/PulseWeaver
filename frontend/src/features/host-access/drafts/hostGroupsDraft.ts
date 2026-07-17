import type { GroupDetailWithUsers, GroupListItem, Id } from "@/lib/api";

export type DraftGroupId = Id | `new-${string}`;

export interface DraftGroup {
  id: DraftGroupId;
  name: string;
  description: string | null;
  icon: string | null;
  color: string;
  hostIds: Id[];
}

export interface GroupsDraftState {
  /** Light metadata for every group (id/name/color/icon/host_count), from the list endpoint. */
  listOriginal: Map<Id, GroupListItem>;
  /** Full detail (description, hosts, users, network_policies), populated lazily per group as it's visited. */
  detailOriginal: Map<Id, GroupDetailWithUsers>;
  /** Group ids whose detailOriginal is loaded and safe to diff/save against. */
  visited: Set<Id>;
  draft: Map<DraftGroupId, DraftGroup>;
  tombstoned: Set<Id>;
  selectedId: DraftGroupId | null;
}

export type GroupsDraftAction =
  | { type: "reset"; groups: GroupListItem[] }
  | { type: "add"; id: `new-${string}`; group: Omit<DraftGroup, "id"> }
  | { type: "update"; id: DraftGroupId; patch: Partial<Omit<DraftGroup, "id">> }
  | { type: "remove"; id: DraftGroupId }
  | { type: "restore"; id: Id }
  | { type: "select"; id: DraftGroupId | null }
  | { type: "toggleHost"; id: DraftGroupId; hostId: Id }
  | { type: "hydrateDetail"; detail: GroupDetailWithUsers }
  | { type: "discard" };

export function initialGroupsDraft(): GroupsDraftState {
  return {
    listOriginal: new Map(),
    detailOriginal: new Map(),
    visited: new Set(),
    draft: new Map(),
    tombstoned: new Set(),
    selectedId: null,
  };
}

export function fromServerGroups(
  groups: GroupListItem[],
): Omit<GroupsDraftState, "selectedId"> {
  const listOriginal = new Map<Id, GroupListItem>();
  const draft = new Map<DraftGroupId, DraftGroup>();
  for (const g of groups) {
    listOriginal.set(g.id, g);
    draft.set(g.id, {
      id: g.id,
      name: g.name,
      // Not present on the light list — filled in once the group's detail is fetched.
      description: null,
      icon: g.icon,
      color: g.color,
      hostIds: [],
    });
  }
  return { listOriginal, detailOriginal: new Map(), visited: new Set(), draft, tombstoned: new Set() };
}

/** Builds a DraftGroup for a tombstoned or discarded id from whatever source data is available — full detail if the group was visited, otherwise the light-list placeholder. */
function draftFromSource(state: GroupsDraftState, id: Id): DraftGroup | null {
  const detail = state.detailOriginal.get(id);
  if (detail) {
    return {
      id,
      name: detail.name,
      description: detail.description ?? null,
      icon: detail.icon,
      color: detail.color,
      hostIds: detail.hosts.map((h) => h.id),
    };
  }
  const listItem = state.listOriginal.get(id);
  if (!listItem) return null;
  return {
    id,
    name: listItem.name,
    description: null,
    icon: listItem.icon,
    color: listItem.color,
    hostIds: [],
  };
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
      const restored = draftFromSource(state, action.id);
      const draft = new Map(state.draft);
      if (restored) draft.set(action.id, restored);
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

    case "hydrateDetail": {
      const { detail } = action;
      const detailOriginal = new Map(state.detailOriginal);
      detailOriginal.set(detail.id, detail);
      const alreadyVisited = state.visited.has(detail.id);
      const visited = new Set(state.visited);
      visited.add(detail.id);
      if (alreadyVisited) {
        // Local edits (metadata or membership) are already staged — a refetch must not clobber them.
        return { ...state, detailOriginal, visited };
      }
      const draft = new Map(state.draft);
      const existing = draft.get(detail.id);
      draft.set(detail.id, {
        ...(existing ?? { id: detail.id, name: detail.name, icon: detail.icon, color: detail.color }),
        description: detail.description ?? null,
        hostIds: detail.hosts.map((h) => h.id),
      });
      return { ...state, detailOriginal, visited, draft };
    }

    case "discard": {
      const draft = new Map<DraftGroupId, DraftGroup>();
      for (const id of state.listOriginal.keys()) {
        const restored = draftFromSource(state, id);
        if (restored) draft.set(id, restored);
      }
      return { ...state, draft, tombstoned: new Set(), selectedId: state.selectedId };
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
  removed: DraftGroupId[];
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
    // Never-visited groups can't be diffed (no ground truth for description/hosts yet);
    // they're also never editable in the UI before their detail loads, so this is safe.
    const original = state.detailOriginal.get(entry.id);
    if (!original) continue;
    const diffEntry = computeGroupDiff(entry, original);
    if (isGroupEntryDirty(diffEntry)) {
      changed.push(diffEntry);
      byId.set(entry.id, diffEntry);
    }
  }

  const removed: DraftGroupId[] = [];
  for (const id of state.tombstoned) {
    if (state.listOriginal.has(id)) {
      removed.push(id);
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
  original: GroupDetailWithUsers,
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
    iconChanged: (draft.icon ?? "") !== original.icon,
    colorChanged: draft.color !== original.color,
    hostsAdded,
    hostsRemoved,
  };
}

/** Builds an editable DraftGroup snapshot from whatever ground truth is available for `id` — used to seed the metadata-edit modal and the tombstoned-row list. */
export function toDraftFromOriginal(state: GroupsDraftState, id: Id): DraftGroup | null {
  return draftFromSource(state, id);
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
