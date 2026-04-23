import { useQuery } from "@tanstack/react-query";
import { listHostGroupsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useHostGroups() {
  return useQuery(listHostGroupsOptions());
}
