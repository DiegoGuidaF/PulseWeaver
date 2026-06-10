import type { SubjectGroupDetail } from "@/lib/api";

export interface SubjectAccessDraft {
  bypassHostCheck: boolean;
  assignedGroupIds: Set<number>;
  bypassAcknowledged: boolean;
}

export type SubjectAccessAction =
  | { type: "reset"; groups: SubjectGroupDetail[]; bypassHostCheck: boolean }
  | { type: "setBypass"; value: boolean }
  | { type: "acknowledgeBypass"; value: boolean }
  | { type: "toggleGroup"; id: number; assigned: boolean };

export function subjectAccessReducer(
  state: SubjectAccessDraft,
  action: SubjectAccessAction,
): SubjectAccessDraft {
  switch (action.type) {
    case "reset":
      return initDraftFromGroups(action.groups, action.bypassHostCheck);
    case "setBypass":
      return { ...state, bypassHostCheck: action.value, bypassAcknowledged: false };
    case "acknowledgeBypass":
      return { ...state, bypassAcknowledged: action.value };
    case "toggleGroup": {
      const next = new Set(state.assignedGroupIds);
      if (action.assigned) next.add(action.id);
      else next.delete(action.id);
      return { ...state, assignedGroupIds: next };
    }
  }
}

export function initialSubjectAccessDraft(): SubjectAccessDraft {
  return { bypassHostCheck: false, assignedGroupIds: new Set(), bypassAcknowledged: false };
}

export function initDraftFromGroups(
  groups: SubjectGroupDetail[],
  bypassHostCheck: boolean,
): SubjectAccessDraft {
  return {
    bypassHostCheck,
    assignedGroupIds: new Set(groups.filter((g) => g.granted).map((g) => g.id)),
    // The "saved" draft always starts acknowledged: a previously-saved bypass
    // state is not a pending action that needs confirming, and a previously-off
    // state has nothing to acknowledge. `isBypassJustEnabled` below is what
    // decides whether the *current* session needs a fresh acknowledgement.
    bypassAcknowledged: true,
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

/**
 * True when the draft turns bypass on while the saved (server) state has it
 * off — the only transition that exposes new blast radius and therefore the
 * only one gated by an explicit acknowledgement. Editing an already-bypassed
 * subject in other ways (group assignments, which are inert while bypass is
 * active, or toggling bypass off) does not require re-confirming a danger the
 * admin already accepted — that would be alert fatigue for no safety gain.
 */
export function isBypassJustEnabled(
  saved: SubjectAccessDraft,
  draft: SubjectAccessDraft,
): boolean {
  return !saved.bypassHostCheck && draft.bypassHostCheck;
}

/**
 * True when the pending change set requires the admin to acknowledge the
 * bypass blast radius before saving is allowed.
 */
export function requiresBypassAcknowledgement(
  saved: SubjectAccessDraft,
  draft: SubjectAccessDraft,
): boolean {
  return isBypassJustEnabled(saved, draft) && !draft.bypassAcknowledged;
}
