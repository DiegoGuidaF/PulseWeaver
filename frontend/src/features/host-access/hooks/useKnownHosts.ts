import { useQuery } from "@tanstack/react-query";
import { listKnownHostsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useKnownHosts() {
  return useQuery(listKnownHostsOptions());
}
