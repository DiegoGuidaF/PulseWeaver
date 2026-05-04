import { useQuery } from "@tanstack/react-query";
import { getPolicyMapOptions } from "@/lib/api/@tanstack/react-query.gen";

export function usePolicyMap() {
  return useQuery(getPolicyMapOptions());
}
