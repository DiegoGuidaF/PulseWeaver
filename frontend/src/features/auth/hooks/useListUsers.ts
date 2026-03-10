import { useQuery } from "@tanstack/react-query";
import { listUsersOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useListUsers(options?: { enabled?: boolean }) {
  return useQuery({
    ...listUsersOptions(),
    enabled: options?.enabled ?? true,
  });
}
