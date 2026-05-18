import type { UpdateNetworkPolicyHostAccessRequest } from "@/lib/api";
import type { NetworkPolicyHostAccessDraft } from "./networkPolicyHostAccessDraft";

export function buildHostAccessBody(
    draft: NetworkPolicyHostAccessDraft,
): UpdateNetworkPolicyHostAccessRequest {
    return {
        allow_all_hosts: draft.allowAllHosts,
        host_group_ids: draft.allowAllHosts ? [] : [...draft.assignedGroupIds],
        host_ids: draft.allowAllHosts ? [] : [...draft.assignedHostIds],
    };
}
