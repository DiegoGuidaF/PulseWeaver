import { useQuery } from "@tanstack/react-query";
import { listUsersWithAccessOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useListUsersWithAccess() {
  return useQuery(listUsersWithAccessOptions());
}
