import { useQuery } from "@tanstack/react-query";
import { getHostGroupOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { Id } from "@/lib/api";

export function useHostGroupDetail(groupId: Id | null) {
  return useQuery({
    ...getHostGroupOptions({ path: { group_id: groupId ?? 0 } }),
    enabled: groupId !== null,
  });
}
