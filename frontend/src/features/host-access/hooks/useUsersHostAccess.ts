import { useQuery } from "@tanstack/react-query";
import { listUsersHostAccessOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useUsersHostAccess() {
  return useQuery(listUsersHostAccessOptions());
}
