import type { NetworkPolicyDetail } from "@/lib/api";

export interface NetworkPolicyHostAccessDraft {
    allowAllHosts: boolean;
    assignedGroupIds: Set<number>;
    assignedHostIds: Set<number>;
}

export type NetworkPolicyHostAccessAction =
    | { type: "reset"; detail: NetworkPolicyDetail }
    | { type: "setAllowAll"; value: boolean }
    | { type: "toggleGroup"; id: number; assigned: boolean }
    | { type: "toggleHost"; id: number; assigned: boolean };

export function networkPolicyHostAccessReducer(
    state: NetworkPolicyHostAccessDraft,
    action: NetworkPolicyHostAccessAction,
): NetworkPolicyHostAccessDraft {
    switch (action.type) {
        case "reset":
            return initDraftFromDetail(action.detail);
        case "setAllowAll":
            return { ...state, allowAllHosts: action.value };
        case "toggleGroup": {
            const next = new Set(state.assignedGroupIds);
            if (action.assigned) next.add(action.id);
            else next.delete(action.id);
            return { ...state, assignedGroupIds: next };
        }
        case "toggleHost": {
            const next = new Set(state.assignedHostIds);
            if (action.assigned) next.add(action.id);
            else next.delete(action.id);
            return { ...state, assignedHostIds: next };
        }
    }
}

export function initialNetworkPolicyHostAccessDraft(): NetworkPolicyHostAccessDraft {
    return { allowAllHosts: false, assignedGroupIds: new Set(), assignedHostIds: new Set() };
}

export function initDraftFromDetail(detail: NetworkPolicyDetail): NetworkPolicyHostAccessDraft {
    return {
        allowAllHosts: detail.allow_all_hosts,
        assignedGroupIds: new Set(detail.host_groups.filter((g) => g.assigned).map((g) => g.id)),
        assignedHostIds: new Set(detail.individual_hosts.filter((h) => h.assigned).map((h) => h.id)),
    };
}

function setsEqual(a: Set<number>, b: Set<number>): boolean {
    if (a.size !== b.size) return false;
    for (const v of a) {
        if (!b.has(v)) return false;
    }
    return true;
}

export function isHostAccessDirty(
    a: NetworkPolicyHostAccessDraft,
    b: NetworkPolicyHostAccessDraft,
): boolean {
    return (
        a.allowAllHosts !== b.allowAllHosts ||
        !setsEqual(a.assignedGroupIds, b.assignedGroupIds) ||
        !setsEqual(a.assignedHostIds, b.assignedHostIds)
    );
}
