import { useQuery } from "@tanstack/react-query";
import { getUserAccessDetailOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useUserAccessDetail(id: number) {
  return useQuery({
    ...getUserAccessDetailOptions({ path: { user_id: id } }),
    enabled: !isNaN(id),
  });
}
