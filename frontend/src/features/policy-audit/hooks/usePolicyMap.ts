import { useQuery } from "@tanstack/react-query";
import { getPolicyUserMapOptions } from "@/lib/api/@tanstack/react-query.gen";

export function usePolicyMap() {
  return useQuery(getPolicyUserMapOptions());
}
