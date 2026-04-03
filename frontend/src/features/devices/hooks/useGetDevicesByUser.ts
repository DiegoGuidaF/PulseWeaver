import { useQuery } from "@tanstack/react-query";
import { getDevicesByUserOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useGetDevicesByUser(userId: number | null) {
  return useQuery({
    ...getDevicesByUserOptions({ path: { user_id: userId ?? 0 } }),
    enabled: userId !== null,
  });
}
