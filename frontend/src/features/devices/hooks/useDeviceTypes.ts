import { useQuery } from "@tanstack/react-query";
import { listDeviceTypesOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useDeviceTypes() {
  return useQuery({
    ...listDeviceTypesOptions(),
    staleTime: Infinity,
  });
}
