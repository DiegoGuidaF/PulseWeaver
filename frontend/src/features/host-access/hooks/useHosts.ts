import { useQuery } from "@tanstack/react-query";
import { listHostsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useHosts() {
  return useQuery(listHostsOptions());
}
