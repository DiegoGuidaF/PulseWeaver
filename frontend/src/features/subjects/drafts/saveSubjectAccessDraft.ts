import type { ModifyAccessRequest } from "@/lib/api";
import type { SubjectAccessDraft } from "./subjectAccessDraft";

export function buildModifyAccessRequest(draft: SubjectAccessDraft): ModifyAccessRequest {
  return {
    bypass_host_check: draft.bypassHostCheck,
    group_ids: [...draft.assignedGroupIds],
  };
}
