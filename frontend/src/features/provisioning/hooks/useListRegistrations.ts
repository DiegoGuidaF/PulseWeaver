import { useQuery } from "@tanstack/react-query";
import { listRegistrationsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useListRegistrations(status: "pending" | "all" = "pending") {
  return useQuery(listRegistrationsOptions({ query: { status } }));
}
