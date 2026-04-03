import { useQuery } from "@tanstack/react-query";
import { getDevicesByUserOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useGetDevicesByUser(userId: number) {
  return useQuery(getDevicesByUserOptions({ path: { user_id: userId } }));
}
