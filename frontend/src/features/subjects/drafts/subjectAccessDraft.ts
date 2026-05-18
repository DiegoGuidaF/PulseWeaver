import type { SubjectGroupDetail } from "@/lib/api";

export interface SubjectAccessDraft {
  bypassHostCheck: boolean;
  assignedGroupIds: Set<number>;
}

export type SubjectAccessAction =
  | { type: "reset"; groups: SubjectGroupDetail[]; bypassHostCheck: boolean }
  | { type: "setBypass"; value: boolean }
  | { type: "toggleGroup"; id: number; assigned: boolean };

export function subjectAccessReducer(
  state: SubjectAccessDraft,
  action: SubjectAccessAction,
): SubjectAccessDraft {
  switch (action.type) {
    case "reset":
      return initDraftFromGroups(action.groups, action.bypassHostCheck);
    case "setBypass":
      return { ...state, bypassHostCheck: action.value };
    case "toggleGroup": {
      const next = new Set(state.assignedGroupIds);
      if (action.assigned) next.add(action.id);
      else next.delete(action.id);
      return { ...state, assignedGroupIds: next };
    }
  }
}

export function initialSubjectAccessDraft(): SubjectAccessDraft {
  return { bypassHostCheck: false, assignedGroupIds: new Set() };
}

export function initDraftFromGroups(
  groups: SubjectGroupDetail[],
  bypassHostCheck: boolean,
): SubjectAccessDraft {
  return {
    bypassHostCheck,
    assignedGroupIds: new Set(groups.filter((g) => g.granted).map((g) => g.id)),
  };
}

function setsEqual(a: Set<number>, b: Set<number>): boolean {
  if (a.size !== b.size) return false;
  for (const v of a) {
    if (!b.has(v)) return false;
  }
  return true;
}

export function isSubjectAccessDirty(
  a: SubjectAccessDraft,
  b: SubjectAccessDraft,
): boolean {
  return (
    a.bypassHostCheck !== b.bypassHostCheck ||
    !setsEqual(a.assignedGroupIds, b.assignedGroupIds)
  );
}
