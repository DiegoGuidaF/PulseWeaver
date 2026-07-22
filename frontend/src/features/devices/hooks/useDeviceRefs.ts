import { useQuery } from "@tanstack/react-query";
import { listDeviceRefsOptions } from "@/lib/api/@tanstack/react-query.gen";

/** Flat {id, name, owner_id} device references, for pickers and the device→owner reverse lookup. */
export function useDeviceRefs() {
  return useQuery(listDeviceRefsOptions());
}
