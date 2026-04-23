import { useQuery } from "@tanstack/react-query";
import { getUserHostDetailsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useUserHostDetails(userId: number | null) {
  return useQuery({
    ...getUserHostDetailsOptions({ path: { user_id: userId ?? 0 } }),
    enabled: userId !== null,
  });
}
