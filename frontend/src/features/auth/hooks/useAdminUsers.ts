import { useQuery } from "@tanstack/react-query";
import { listUsersOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useAdminUsers() {
  return useQuery({
    ...listUsersOptions(),
    staleTime: 60_000,
  });
}
